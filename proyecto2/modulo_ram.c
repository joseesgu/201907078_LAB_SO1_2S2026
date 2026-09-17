#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>

#define PROC_NAME "continfo_pr2_so1_201907078"

static int escribir_archivo_proc(struct seq_file *m, void *v) {
    struct sysinfo i;
    struct task_struct *task;
    
    // 1. OBTENER INFORMACIÓN DE LA MEMORIA
    si_meminfo(&i);
    long total_ram = (i.totalram * i.mem_unit) / 1024;     // En KB
    long free_ram = (i.freeram * i.mem_unit) / 1024;       // En KB
    long used_ram = total_ram - free_ram;                  // En KB

    seq_printf(m, "--- MÉTRICAS DE MEMORIA ---\n");
    seq_printf(m, "Total RAM: %ld KB\n", total_ram); //
    seq_printf(m, "Free RAM: %ld KB\n", free_ram); //[cite: 1]
    seq_printf(m, "Used RAM: %ld KB\n", used_ram); //[cite: 1]
    
    seq_printf(m, "\n--- LISTADO DE PROCESOS ---\n");
    // Añadimos las columnas de %MEM y %CPU[cite: 1]
    seq_printf(m, "PID\tNombre\tVSZ(KB)\tRSS(KB)\t%%MEM\t%%CPU\tCMDLINE\n");
    
    // 2. ITERAR SOBRE LOS PROCESOS
    for_each_process(task) {
        long vsz = 0;
        long rss = 0;
        long mem_porcentaje = 0;
        unsigned long cpu_time = 0;
        char cmdline[256]; 
        int res = 0;
        
        memset(cmdline, 0, sizeof(cmdline));
        
        // Tiempos de CPU acumulados (jiffies)
        cpu_time = task->utime + task->stime;
        
        // Verificamos que el proceso tenga un espacio de memoria asignado
        if (task->mm) {
            vsz = (task->mm->total_vm * PAGE_SIZE) / 1024; 
            rss = (get_mm_rss(task->mm) * PAGE_SIZE) / 1024;
            
            if (total_ram > 0) {
                mem_porcentaje = (rss * 100) / total_ram;
            }
            
            // Leer la línea de comando
            if (task->mm->arg_end != 0 && task->mm->arg_start != 0) {
                unsigned long len = task->mm->arg_end - task->mm->arg_start;
                if (len > sizeof(cmdline) - 1) {
                    len = sizeof(cmdline) - 1;
                }
                res = access_process_vm(task, task->mm->arg_start, cmdline, len, 0);
            }
        }
        
        // Formatear la línea de comando
        if (res > 0) {
            for (int j = 0; j < res - 1; j++) {
                if (cmdline[j] == '\0') {
                    cmdline[j] = ' ';
                }
            }
        } else {
            strcpy(cmdline, "N/A");
        }
        
        // Imprimir todos los datos requeridos[cite: 1]
        seq_printf(m, "%d\t%s\t%ld\t%ld\t%ld%%\t%lu\t%s\n", 
                   task->pid, task->comm, vsz, rss, mem_porcentaje, cpu_time, cmdline);
    }
    
    return 0;
}

static int al_abrir(struct inode *inode, struct file *file) {
    return single_open(file, escribir_archivo_proc, NULL);
}

static const struct proc_ops operaciones = {
    .proc_open = al_abrir,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init modulo_init(void) {
    proc_create(PROC_NAME, 0, NULL, &operaciones);
    printk(KERN_INFO "Modulo cargado exitosamente. Archivo creado en /proc/%s\n", PROC_NAME);
    return 0;
}

static void __exit modulo_cleanup(void) {
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "Modulo removido. Archivo /proc/%s eliminado\n", PROC_NAME);
}

module_init(modulo_init);
module_exit(modulo_cleanup);

MODULE_LICENSE("GPL");
MODULE_DESCRIPTION("Sonda de kernel para metricas de contenedores");
MODULE_AUTHOR("Jose Estrada - 201907078");