#!/usr/bin/python3
from bcc import BPF
import sys

bpf_text = """
#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

// Interceptar sys_kill de forma estructurada
TRACEPOINT_PROBE(syscalls, sys_enter_kill) {
    if (args->sig == 9 || args->sig == 15) {
        bpf_trace_printk("EBPF_KILL:%d:%d\\n", args->pid, args->sig);
    }
    return 0;
}

// Interceptar sys_tgkill (por si Docker usa sub-hilos)
TRACEPOINT_PROBE(syscalls, sys_enter_tgkill) {
    if (args->sig == 9 || args->sig == 15) {
        bpf_trace_printk("EBPF_KILL:%d:%d\\n", args->pid, args->sig);
    }
    return 0;
}
"""
try:
    b = BPF(text=bpf_text)
    print("AUDITOR_READY", flush=True)
except Exception as e:
    sys.exit(1)

while True:
    try:
        (task, pid, cpu, flags, ts, msg) = b.trace_fields()
        mensaje = msg.decode('utf-8')
        if "EBPF_KILL:" in mensaje:
            partes = mensaje.split("EBPF_KILL:")
            print("EBPF_KILL:" + partes[1].strip(), flush=True)
    except KeyboardInterrupt:
        sys.exit(0)
    except Exception:
        continue
