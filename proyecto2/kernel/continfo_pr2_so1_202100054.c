#include <linux/init.h>
#include <linux/kernel.h>
#include <linux/mm.h>
#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sysinfo.h>

#define PROC_NAME "continfo_pr2_so1_202100054"

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Victor Hugo Velasquez Hernandez");
MODULE_DESCRIPTION("Telemetria de memoria y procesos para Proyecto 2 SO1");
MODULE_VERSION("1.0");

static struct proc_dir_entry *proc_entry;

static unsigned long long pages_to_kb(unsigned long pages)
{
    return (unsigned long long)pages << (PAGE_SHIFT - 10);
}

static int continfo_show(struct seq_file *file, void *private_data)
{
    struct sysinfo memory_info;
    unsigned long long total_kb;
    unsigned long long free_kb;
    unsigned long long used_kb;

    si_meminfo(&memory_info);

    total_kb = pages_to_kb(memory_info.totalram);
    free_kb = pages_to_kb(memory_info.freeram);
    used_kb = total_kb - free_kb;

    seq_puts(file, "{\n");
    seq_puts(file, "  \"memory\": {\n");
    seq_printf(file, "    \"total_kb\": %llu,\n", total_kb);
    seq_printf(file, "    \"free_kb\": %llu,\n", free_kb);
    seq_printf(file, "    \"used_kb\": %llu\n", used_kb);
    seq_puts(file, "  },\n");
    seq_puts(file, "  \"processes\": []\n");
    seq_puts(file, "}\n");

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
    proc_entry = proc_create(
        PROC_NAME,
        0444,
        NULL,
        &continfo_proc_ops
    );

    if (!proc_entry) {
        pr_err("SO1 Proyecto 2: no se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }

    pr_info("SO1 Proyecto 2: modulo cargado, /proc/%s creado\n", PROC_NAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    proc_remove(proc_entry);
    pr_info("SO1 Proyecto 2: modulo retirado, /proc/%s eliminado\n", PROC_NAME);
}

module_init(continfo_init);
module_exit(continfo_exit);