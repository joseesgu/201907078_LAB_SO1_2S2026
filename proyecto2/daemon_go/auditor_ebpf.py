#!/usr/bin/python3
from bcc import BPF
import sys

bpf_text = """
#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

// Interceptar sys_kill estándar[cite: 1]
int kprobe__sys_kill(struct pt_regs *ctx, pid_t pid, int sig) {
    if (sig == 9 || sig == 15) {
        bpf_trace_printk("EBPF_KILL:%d:%d\\n", pid, sig);
    }
    return 0;
}

// Interceptar sys_tgkill (Thread Group Kill)
int kprobe__sys_tgkill(struct pt_regs *ctx, pid_t tgid, pid_t pid, int sig) {
    if (sig == 9 || sig == 15) {
        bpf_trace_printk("EBPF_KILL:%d:%d\\n", pid, sig);
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
