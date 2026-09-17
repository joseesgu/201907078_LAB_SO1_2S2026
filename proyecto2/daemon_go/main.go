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
)

// Proceso almacena la información extraída del kernel
type Proceso struct {
	PID     int
	Nombre  string
	VSZ     int
	RSS     int
	MemPct  string
	CPU     int
	Cmdline string
}

// Contenedor unifica los datos de Docker con las métricas del Kernel[cite: 1]
type Contenedor struct {
	ID     string
	Nombre string
	PID    int
	VSZ    int
	RSS    int
	CPU    int
}

var ctx = context.Background()
var rdb *redis.Client

// --- FUNCIONES DE AUTOMATIZACIÓN E INFRAESTRUCTURA ---

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
	fmt.Println("[INFO] Configurando cronjob de despliegue cada 2 minutos...")
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
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
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
		if strings.HasPrefix(linea, "PID") {
			leyendoProcesos = true
			continue
		}

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

// --- TOMA DE DECISIONES Y GESTIÓN ---

func gestionarContenedores(contenedores []Contenedor) {
	var altos, bajos, eliminar []Contenedor

	// 1. Categorizar contenedores
	for _, c := range contenedores {
		// Restricción: No debe eliminarse el contenedor de Grafana (ni Valkey)[cite: 1]
		if strings.Contains(c.Nombre, "grafana") || strings.Contains(c.Nombre, "valkey") {
			continue
		}

		if strings.Contains(c.Nombre, "alto-ram") || strings.Contains(c.Nombre, "alto-cpu") {
			altos = append(altos, c)
		} else if strings.Contains(c.Nombre, "bajo-consumo") {
			bajos = append(bajos, c)
		} else {
			// El contenedor Intruso o cualquier no clasificado se marca para eliminación[cite: 1]
			eliminar = append(eliminar, c)
		}
	}

	// 2. Aplicar ordenamiento basado en consumo (Prioridad a RSS, luego CPU)[cite: 1]
	// Ordenamos de mayor a menor para eliminar a los que más recursos acaparan
	ordenarPorConsumo := func(lista []Contenedor) {
		sort.Slice(lista, func(i, j int) bool {
			if lista[i].RSS != lista[j].RSS {
				return lista[i].RSS > lista[j].RSS // Mayor RSS primero
			}
			return lista[i].CPU > lista[j].CPU // Mayor CPU en caso de empate
		})
	}

	ordenarPorConsumo(altos)
	ordenarPorConsumo(bajos)

	// 3. Evaluar reglas y seleccionar sobrantes
	// Siempre deben existir 2 contenedores de alto consumo[cite: 1]
	if len(altos) > 2 {
		eliminar = append(eliminar, altos[2:]...) // Del 3ro en adelante, a la guillotina
	}
	// Siempre deben existir 3 contenedores de bajo consumo[cite: 1]
	if len(bajos) > 3 {
		eliminar = append(eliminar, bajos[3:]...)
	}

	// 4. Ejecutar decisiones
	if len(eliminar) == 0 {
		fmt.Println("[GESTIÓN] El sistema está equilibrado. No se requiere eliminar contenedores.")
		return
	}

	for _, c := range eliminar {
		fmt.Printf("[ACCIÓN] Deteniendo y eliminando contenedor: %s (PID: %d)\n", c.Nombre, c.PID)

		// Utilizamos el CLI de Docker para forzar la detención y eliminación (SIGKILL)[cite: 1]
		// Esto desencadenará el evento sys_kill que interceptará eBPF[cite: 1]
		err := exec.Command("docker", "rm", "-f", c.ID).Run()

		if err != nil {
			fmt.Printf("[ERROR] Fallo al eliminar contenedor %s: %v\n", c.Nombre, err)
		}
	}
}

// Configura la conexión a la base de datos Valkey
func conectarValkey() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Puerto expuesto en tu docker-compose[cite: 1]
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("[ERROR] No se pudo conectar a Valkey: %v\n", err)
	} else {
		fmt.Println("[INFO] Conectado a Valkey exitosamente.")
	}
}

// Lanza el script de Python y escucha sus impresiones en tiempo real
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
			// Confirmación real a nivel de kernel de que el contenedor fue terminado[cite: 1]
			partes := strings.Split(linea, ":")
			if len(partes) >= 3 {
				fmt.Printf("[eBPF] Intercepción confirmada -> Señal %s enviada al PID %s\n", partes[2], partes[1])
				// Solo en este momento registramos el evento en Valkey[cite: 1]
				rdb.Incr(ctx, "total_contenedores_eliminados")
			}
		}
	}
}

// --- FUNCIÓN PRINCIPAL ---

func main() {
	fmt.Println("=== Iniciando Daemon Gestor de Contenedores ===")

	levantarInfraestructura()
	time.Sleep(3 * time.Second) // Breve pausa para que Valkey inicie completamente
	conectarValkey()
	go iniciarAuditorEBPF() // Se ejecuta en segundo plano
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
					c := Contenedor{
						ID:     datosDocker.ID,
						Nombre: datosDocker.Nombre,
						PID:    proc.PID,
						VSZ:    proc.VSZ,
						RSS:    proc.RSS,
						CPU:    proc.CPU,
					}
					contenedoresActivos = append(contenedoresActivos, c)
				}
			}

			fmt.Printf("[FILTRO] Se aislaron %d procesos de contenedores Docker.\n", len(contenedoresActivos))

			// --- RESTAURACIÓN DE LA TABLA VISUAL ---
			fmt.Println("ID_CONTENEDOR\t\tNOMBRE\t\tPID\tVSZ\tRSS\tCPU")
			for _, c := range contenedoresActivos {
				idCorto := c.ID
				if len(idCorto) > 12 {
					idCorto = idCorto[:12]
				}
				fmt.Printf("%s\t%s\t%d\t%d\t%d\t%d\n", idCorto, c.Nombre, c.PID, c.VSZ, c.RSS, c.CPU)
			}
			fmt.Println("---------------------------------------------------------")
			// ---------------------------------------

			// Llamar a la lógica de toma de decisiones
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
