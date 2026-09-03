#include <linux/init.h>
#include <linux/kernel.h>
#include <linux/math64.h>
#include <linux/mm.h>
#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/rcupdate.h>
#include <linux/sched/mm.h>
#include <linux/sched/signal.h>
#include <linux/seq_file.h>
#include <linux/sysinfo.h>

#include <linux/ktime.h>
#include <linux/sched/cputime.h>

#define PROC_NAME "continfo_pr2_so1_202100054"

static struct proc_dir_entry *proc_entry;

static void print_json_string(struct seq_file *m, const char *text)
{
    const unsigned char *cursor = (const unsigned char *)text;

    seq_putc(m, '"');

    while (*cursor) {
        switch (*cursor) {
        case '"':
            seq_puts(m, "\\\"");
            break;
        case '\\':
            seq_puts(m, "\\\\");
            break;
        case '\n':
            seq_puts(m, "\\n");
            break;
        case '\r':
            seq_puts(m, "\\r");
            break;
        case '\t':
            seq_puts(m, "\\t");
            break;
        default:
            if (*cursor < 0x20)
                seq_printf(m, "\\u%04x", *cursor);
            else
                seq_putc(m, *cursor);
        }

        cursor++;
    }

    seq_putc(m, '"');
}



static void print_process(struct seq_file *m,
                          struct task_struct *task,
                          u64 total_memory_kb,
                          bool *first_process)
{
    struct mm_struct *mm;
    char name[TASK_COMM_LEN];
    u64 virtual_size_kb;
    u64 resident_size_kb;
    u64 memory_basis_points;
    u64 user_time_ns;
    u64 system_time_ns;
    u64 cpu_time_ms;
    u64 elapsed_time_ms;
    u64 start_boottime_ns;
    u64 current_boottime_ns;
    u64 cpu_basis_points;

    mm = get_task_mm(task);
    if (!mm)
        return;

    virtual_size_kb =
        ((u64)READ_ONCE(mm->total_vm)) << (PAGE_SHIFT - 10);

    resident_size_kb =
        ((u64)get_mm_rss(mm)) << (PAGE_SHIFT - 10);

    mmput(mm);

    get_task_comm(name, task);



    task_cputime_adjusted(task , &user_time_ns, &system_time_ns);

    start_boottime_ns = READ_ONCE(task->start_boottime);
    current_boottime_ns = ktime_get_boottime_ns();

    cpu_time_ms = 
        div_u64(user_time_ns + system_time_ns, 1000000);
    
    if (current_boottime_ns > start_boottime_ns) {
        elapsed_time_ms =
            div_u64(current_boottime_ns - start_boottime_ns,
                    NSEC_PER_MSEC);
    } else {
        elapsed_time_ms = 0;
    }

    if (elapsed_time_ms > 0)
        cpu_basis_points =
            div64_u64(cpu_time_ms * 10000, elapsed_time_ms);
    else
        cpu_basis_points = 0;

    /* Un proceso individual no debe superar el 100 %. */
    if (cpu_basis_points > 10000)
        cpu_basis_points = 10000;

    if (total_memory_kb > 0)
        memory_basis_points =
            div64_u64(resident_size_kb * 10000, total_memory_kb);
    else
        memory_basis_points = 0;

    if (!*first_process)
        seq_puts(m, ",\n");

    *first_process = false;

    seq_puts(m, "    {\n");

    seq_printf(m, "      \"pid\": %d,\n", task_pid_nr(task));

    seq_puts(m, "      \"name\": ");
    print_json_string(m, name);
    seq_puts(m, ",\n");

    /*
     * task->comm es seguro para módulos externos. El Daemon en Go
     * enriquecerá este dato con el ID real del contenedor.
     */
    seq_puts(m, "      \"command\": ");
    print_json_string(m, name);
    seq_puts(m, ",\n");

    seq_printf(m, "      \"vsz_kb\": %llu,\n",
               (unsigned long long)virtual_size_kb);

    seq_printf(m, "      \"rss_kb\": %llu,\n",
               (unsigned long long)resident_size_kb);

    seq_printf(m, "      \"memory_percent\": %llu.%02llu,\n",
               (unsigned long long)(memory_basis_points / 100),
               (unsigned long long)(memory_basis_points % 100));

    seq_printf(m, "      \"cpu_percent\": %llu.%02llu\n",
               (unsigned long long)(cpu_basis_points / 100),
               (unsigned long long)(cpu_basis_points % 100));

    seq_puts(m, "    }");
}

static int continfo_show(struct seq_file *m, void *v)
{
    struct sysinfo memory_info;
    struct task_struct *task;
    u64 total_memory_kb;
    u64 free_memory_kb;
    u64 used_memory_kb;
    bool first_process = true;

    si_meminfo(&memory_info);

    total_memory_kb =
        div_u64((u64)memory_info.totalram * memory_info.mem_unit, 1024);

    free_memory_kb =
        div_u64((u64)memory_info.freeram * memory_info.mem_unit, 1024);

    used_memory_kb = total_memory_kb - free_memory_kb;

    seq_puts(m, "{\n");
    seq_puts(m, "  \"memory\": {\n");
    seq_printf(m, "    \"total_kb\": %llu,\n",
               (unsigned long long)total_memory_kb);
    seq_printf(m, "    \"free_kb\": %llu,\n",
               (unsigned long long)free_memory_kb);
    seq_printf(m, "    \"used_kb\": %llu\n",
               (unsigned long long)used_memory_kb);
    seq_puts(m, "  },\n");
    seq_puts(m, "  \"processes\": [\n");

    /*
     * Se conserva una referencia por proceso para poder consultar su
     * memoria y línea de comandos fuera de la sección crítica RCU.
     */
    rcu_read_lock();

    for_each_process(task) {
        get_task_struct(task);
        rcu_read_unlock();

        print_process(m, task, total_memory_kb, &first_process);

        rcu_read_lock();
        put_task_struct(task);
    }

    rcu_read_unlock();

    seq_puts(m, "\n  ]\n");
    seq_puts(m, "}\n");

    return 0;
}

static int continfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, continfo_show, NULL);
}

static const struct proc_ops continfo_proc_ops = {
    .proc_open = continfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init continfo_init(void)
{
    proc_entry = proc_create(PROC_NAME, 0444, NULL, &continfo_proc_ops);

    if (!proc_entry) {
        pr_err("SO1 Proyecto 2: no se pudo crear /proc/%s\n",
               PROC_NAME);
        return -ENOMEM;
    }

    pr_info("SO1 Proyecto 2: modulo cargado, /proc/%s creado\n",
            PROC_NAME);

    return 0;
}

static void __exit continfo_exit(void)
{
    proc_remove(proc_entry);

    pr_info("SO1 Proyecto 2: modulo retirado, /proc/%s eliminado\n",
            PROC_NAME);
}

module_init(continfo_init);
module_exit(continfo_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Victor Hugo Velasquez Hernandez");
MODULE_DESCRIPTION("Telemetria de memoria y procesos para Proyecto 2 SO1");
MODULE_VERSION("1.2");