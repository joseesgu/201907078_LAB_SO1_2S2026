package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// Variables globales para la conexión a Valkey
var ctx = context.Background()
var rdb *redis.Client

type Proceso struct {
	PID     int
	Nombre  string
	VSZ     int
	RSS     int
	MemPct  string
	CPU     int
	Cmdline string
}

type Contenedor struct {
	ID     string
	Nombre string
	PID    int
	VSZ    int
	RSS    int
	CPU    int
}

// --- CONEXIONES Y AUDITORÍA ---

func conectarValkey() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("[ERROR] No se pudo conectar a Valkey: %v\n", err)
	} else {
		fmt.Println("[INFO] Conectado a Valkey exitosamente.")
	}
}

func iniciarAuditorEBPF() {
	fmt.Println("[INFO] Iniciando sonda eBPF en el kernel...")
	cmd := exec.Command("python3", "auditor_ebpf.py")
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		linea := scanner.Text()
		if linea == "AUDITOR_READY" {
			fmt.Println("[INFO] eBPF Auditor está escuchando syscalls (sys_kill).")
		} else if strings.HasPrefix(linea, "EBPF_KILL:") {
			partes := strings.Split(linea, ":")
			if len(partes) >= 3 {
				fmt.Printf("[eBPF] Intercepción confirmada -> Señal %s enviada al PID %s\n", partes[2], partes[1])
				if rdb != nil {
					rdb.Incr(ctx, "total_contenedores_eliminados")
				}
			}
		}
	}
}

// --- AUTOMATIZACIÓN DEL SISTEMA ---

func cargarModulo() {
	fmt.Println("[INFO] Preparando entorno del kernel...")
	exec.Command("sudo", "rmmod", "modulo_ram").Run()
	fmt.Println("[INFO] Cargando módulo del kernel...")
	cmd := exec.Command("sudo", "insmod", "../modulo_ram.ko")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func descargarModulo() {
	fmt.Println("[INFO] Descargando módulo del kernel...")
	exec.Command("sudo", "rmmod", "modulo_ram").Run()
}

func configurarCron() {
	fmt.Println("[INFO] Configurando cronjob de despliegue...")
	rutaActual, _ := os.Getwd()
	rutaScript := rutaActual + "/../despliegue_aleatorio.sh"
	cronJob := fmt.Sprintf("*/2 * * * * %s\n", rutaScript)
	os.WriteFile("temp_cron", []byte(cronJob), 0644)
	exec.Command("crontab", "temp_cron").Run()
	os.Remove("temp_cron")
}

func eliminarCron() {
	fmt.Println("[INFO] Eliminando cronjob del sistema...")
	exec.Command("crontab", "-r").Run()
}

func levantarInfraestructura() {
	fmt.Println("[INFO] Levantando entorno de Grafana y Valkey...")
	exec.Command("docker", "compose", "up", "-d").Run()
}

func detenerInfraestructura() {
	fmt.Println("[INFO] Deteniendo entorno de Grafana y Valkey...")
	exec.Command("docker", "compose", "down").Run()
}

// --- EXTRACCIÓN DE DATOS ---

func leerArchivoProc() []Proceso {
	archivo, err := os.Open("/proc/continfo_pr2_so1_201907078")
	if err != nil {
		return nil
	}
	defer archivo.Close()

	scanner := bufio.NewScanner(archivo)
	leyendoProcesos := false
	var procesos []Proceso

	for scanner.Scan() {
		linea := scanner.Text()

		// 1. Capturar RAM global para Grafana
		if strings.HasPrefix(linea, "Total RAM:") {
			partes := strings.Fields(linea)
			if len(partes) >= 3 && rdb != nil {
				rdb.Set(ctx, "total_ram", partes[2], 0)
			}
		} else if strings.HasPrefix(linea, "Used RAM:") {
			partes := strings.Fields(linea)
			if len(partes) >= 3 && rdb != nil {
				rdb.Set(ctx, "used_ram", partes[2], 0)
				timestamp := time.Now().Unix()
				rdb.ZAdd(ctx, "ram_historico", redis.Z{Score: float64(timestamp), Member: partes[2]})
			}
		} else if strings.HasPrefix(linea, "Free RAM:") {
			partes := strings.Fields(linea)
			if len(partes) >= 3 && rdb != nil {
				rdb.Set(ctx, "free_ram", partes[2], 0)
			}
		}

		// 2. Transición al listado de procesos
		if strings.HasPrefix(linea, "PID") {
			leyendoProcesos = true
			continue
		}

		// 3. Capturar métricas de cada proceso
		if leyendoProcesos && strings.TrimSpace(linea) != "" {
			campos := strings.Split(linea, "\t")
			if len(campos) >= 7 {
				pid, _ := strconv.Atoi(campos[0])
				vsz, _ := strconv.Atoi(campos[2])
				rss, _ := strconv.Atoi(campos[3])
				cpu, _ := strconv.Atoi(campos[5])

				procesos = append(procesos, Proceso{
					PID: pid, Nombre: campos[1], VSZ: vsz, RSS: rss,
					MemPct: campos[4], CPU: cpu, Cmdline: campos[6],
				})
			}
		}
	}
	return procesos
}

func mapearContenedoresDocker() map[int]struct{ ID, Nombre string } {
	mapeo := make(map[int]struct{ ID, Nombre string })
	out, err := exec.Command("docker", "ps", "-q").Output()
	if err != nil || len(out) == 0 {
		return mapeo
	}

	ids := strings.Fields(string(out))
	args := append([]string{"inspect", "--format", "{{.State.Pid}} {{.Id}} {{.Name}}"}, ids...)
	outInspect, _ := exec.Command("docker", args...).Output()

	for _, linea := range strings.Split(strings.TrimSpace(string(outInspect)), "\n") {
		partes := strings.SplitN(linea, " ", 3)
		if len(partes) == 3 {
			pid, _ := strconv.Atoi(partes[0])
			mapeo[pid] = struct{ ID, Nombre string }{ID: partes[1], Nombre: strings.TrimPrefix(partes[2], "/")}
		}
	}
	return mapeo
}

// --- TOMA DE DECISIONES ---

func gestionarContenedores(contenedores []Contenedor) {
	var altos, bajos, eliminar []Contenedor

	for _, c := range contenedores {
		if strings.Contains(c.Nombre, "grafana") || strings.Contains(c.Nombre, "valkey") {
			continue
		}
		if strings.Contains(c.Nombre, "alto-ram") || strings.Contains(c.Nombre, "alto-cpu") {
			altos = append(altos, c)
		} else if strings.Contains(c.Nombre, "bajo-consumo") {
			bajos = append(bajos, c)
		} else {
			eliminar = append(eliminar, c)
		}
	}

	ordenarPorConsumo := func(lista []Contenedor) {
		sort.Slice(lista, func(i, j int) bool {
			if lista[i].RSS != lista[j].RSS {
				return lista[i].RSS > lista[j].RSS
			}
			return lista[i].CPU > lista[j].CPU
		})
	}

	ordenarPorConsumo(altos)
	ordenarPorConsumo(bajos)

	if len(altos) > 2 {
		eliminar = append(eliminar, altos[2:]...)
	}
	if len(bajos) > 3 {
		eliminar = append(eliminar, bajos[3:]...)
	}

	if len(eliminar) == 0 {
		fmt.Println("[GESTIÓN] El sistema está equilibrado. No se requiere eliminar contenedores.")
	} else {
		for _, c := range eliminar {
			fmt.Printf("[ACCIÓN] Enviando SIGKILL al proceso %d (Contenedor: %s)\n", c.PID, c.Nombre)

			syscall.Kill(c.PID, syscall.SIGKILL)

			// --- PAUSA ESTRATÉGICA ---
			time.Sleep(500 * time.Millisecond)

			exec.Command("docker", "rm", "-f", c.ID).Run()
		}
	}

	// Registrar los Top 5 en Valkey para Grafana
	if rdb != nil {
		sort.Slice(contenedores, func(i, j int) bool { return contenedores[i].RSS > contenedores[j].RSS })
		rdb.Del(ctx, "top_ram")
		for i, c := range contenedores {
			if i >= 5 {
				break
			}
			rdb.HSet(ctx, "top_ram", fmt.Sprintf("PID:%d-ID:%s", c.PID, c.ID[:8]), c.RSS)
		}

		sort.Slice(contenedores, func(i, j int) bool { return contenedores[i].CPU > contenedores[j].CPU })
		rdb.Del(ctx, "top_cpu")
		for i, c := range contenedores {
			if i >= 5 {
				break
			}
			rdb.HSet(ctx, "top_cpu", fmt.Sprintf("PID:%d-ID:%s", c.PID, c.ID[:8]), c.CPU)
		}
	}
}

// --- FUNCIÓN PRINCIPAL ---

func main() {
	fmt.Println("=== Iniciando Daemon Gestor de Contenedores ===")

	levantarInfraestructura()
	time.Sleep(3 * time.Second) // Esperar que la DB esté lista
	conectarValkey()
	go iniciarAuditorEBPF()

	cargarModulo()
	configurarCron()

	canalSalida := make(chan os.Signal, 1)
	signal.Notify(canalSalida, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	fmt.Println("[INFO] Loop principal iniciado. Esperando iteraciones (20s)...")

	for {
		select {
		case <-ticker.C:
			fmt.Println("\n--- Ejecutando iteración de análisis ---")
			procesosKernel := leerArchivoProc()
			mapeoDocker := mapearContenedoresDocker()

			var contenedoresActivos []Contenedor
			for _, proc := range procesosKernel {
				if datosDocker, existe := mapeoDocker[proc.PID]; existe {
					contenedoresActivos = append(contenedoresActivos, Contenedor{
						ID:     datosDocker.ID,
						Nombre: datosDocker.Nombre,
						PID:    proc.PID,
						VSZ:    proc.VSZ,
						RSS:    proc.RSS,
						CPU:    proc.CPU,
					})
				}
			}

			fmt.Printf("[FILTRO] Se aislaron %d procesos de contenedores Docker.\n", len(contenedoresActivos))

			fmt.Println("ID_CONTENEDOR\t\tNOMBRE\t\tPID\tVSZ\tRSS\tCPU")
			for _, c := range contenedoresActivos {
				idCorto := c.ID
				if len(idCorto) > 12 {
					idCorto = idCorto[:12]
				}
				fmt.Printf("%s\t%s\t%d\t%d\t%d\t%d\n", idCorto, c.Nombre, c.PID, c.VSZ, c.RSS, c.CPU)
			}
			fmt.Println("---------------------------------------------------------")

			gestionarContenedores(contenedoresActivos)

		case <-canalSalida:
			fmt.Println("\n[INFO] Señal de apagado recibida (Graceful Shutdown)...")
			eliminarCron()
			descargarModulo()
			detenerInfraestructura()
			fmt.Println("[INFO] Limpieza completada. Apagando servicio.")
			os.Exit(0)
		}
	}
}
