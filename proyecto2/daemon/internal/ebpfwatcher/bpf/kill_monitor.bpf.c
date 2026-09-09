#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

#define SYS_KILL_X86_64 62
#define TASK_COMM_LEN 16

char LICENSE[] SEC("license") = "GPL";

struct kill_event {
    __u64 timestamp_ns;
    __u32 caller_pid;
    __s32 target_pid;
    __s32 signal;
    char command[TASK_COMM_LEN];
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("tracepoint/raw_syscalls/sys_enter")
int trace_kill(struct trace_event_raw_sys_enter *context)
{
    struct kill_event *event;
    __u64 pid_tgid;
    __s32 signal;

    if (context->id != SYS_KILL_X86_64)
        return 0;

    signal = (__s32)context->args[1];

    if (signal <= 0 || signal > 64)
        return 0;

    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    pid_tgid = bpf_get_current_pid_tgid();

    event->timestamp_ns = bpf_ktime_get_ns();
    event->caller_pid = pid_tgid >> 32;
    event->target_pid = (__s32)context->args[0];
    event->signal = signal;

    bpf_get_current_comm(
        event->command,
        sizeof(event->command)
    );

    bpf_ringbuf_submit(event, 0);

    return 0;
}