# cgar

## What Is It?

Do you know `sar`? If so, you get a pretty good idea what `cgar` shall be!

Instead of collecting system resource data, `cgar` will collect data of cgroups.
With `systemd` already partitioning user processes and services into a defined hierarchy, the data is already there and just needs to be collected and processed.

The data can be used to track and investigate resource consumption.

I have put together some examples in [`examples/`](examples/) to show for what `cgar` can be useful.

**This project is in an early stage and has limited support for cgroup controllers. Depending on the reception it made grow in the future.**

## What Do You Need?

1. Your processes have to be grouped in a meaningful way into cgroups. With `systemd` this already should be taken care of.
   Therefore: Check mark! \
   _If you are using one of the rare distributions without `systemd`, you have to build up your own hierarchy first._

2. Only cgroup2 is supported. \
   This should be the default for any recent distribution anyways. Some more conservative or enterprise grade distros still using a hybrid hierarchy where cgroup1 is used primarily. Such a setup is useless for `cgar`, you have to switch to cgroup2-only, also called: unified hierarchy.  \
   Luckily this is done easily and should not lead to any problems, but check first.

3. Depending on your system setup, systemd might not have enabled all the cgroup accounting which is possible.
   If you are missing something, you have to enable it.

4. Depending on your system you need some additional work to have the full set of data in the cgroups. \
   On some systems you need to enable swap accounting as well as pressure stall information. How dto do this will be described later.

After summarizing what do you need, now in more detail how you get it done!


### Switching To Cgroup2

If a `mount | grep cgroup` returns such a line:

```
cgroup2 on /sys/fs/cgroup type cgroup2 (rw,nosuid,nodev,noexec,relatime,nsdelegate,memory_recursiveprot)
```

your are fine. If you find instead something like this: 

```
cgroup2 on /sys/fs/cgroup/unified type cgroup2 (rw,nosuid,nodev,noexec,relatime,nsdelegate)
cgroup on /sys/fs/cgroup/systemd type cgroup (rw,nosuid,nodev,noexec,relatime,xattr,name=systemd)
cgroup on /sys/fs/cgroup/net_cls,net_prio type cgroup (rw,nosuid,nodev,noexec,relatime,net_cls,net_prio)
cgroup on /sys/fs/cgroup/cpuset type cgroup (rw,nosuid,nodev,noexec,relatime,cpuset)
cgroup on /sys/fs/cgroup/pids type cgroup (rw,nosuid,nodev,noexec,relatime,pids)
cgroup on /sys/fs/cgroup/perf_event type cgroup (rw,nosuid,nodev,noexec,relatime,perf_event)
cgroup on /sys/fs/cgroup/rdma type cgroup (rw,nosuid,nodev,noexec,relatime,rdma)
cgroup on /sys/fs/cgroup/memory type cgroup (rw,nosuid,nodev,noexec,relatime,memory)
cgroup on /sys/fs/cgroup/devices type cgroup (rw,nosuid,nodev,noexec,relatime,devices)
cgroup on /sys/fs/cgroup/cpu,cpuacct type cgroup (rw,nosuid,nodev,noexec,relatime,cpu,cpuacct)
cgroup on /sys/fs/cgroup/freezer type cgroup (rw,nosuid,nodev,noexec,relatime,freezer)
cgroup on /sys/fs/cgroup/hugetlb type cgroup (rw,nosuid,nodev,noexec,relatime,hugetlb)
cgroup on /sys/fs/cgroup/blkio type cgroup (rw,nosuid,nodev,noexec,relatime,blkio)
cgroup on /sys/fs/cgroup/misc type cgroup (rw,nosuid,nodev,noexec,relatime,misc)
```

then you must act.

All you have to do, is to tell your boot lader to add `systemd.unified_cgroup_hierarchy=1` to the kernel command line.
On OpenLeap or SLES simply open `/etc/default/grub` with an editor and alter `GRUB_CMDLINE_LINUX_DEFAULT`:

```
GRUB_CMDLINE_LINUX_DEFAULT="... systemd.unified_cgroup_hierarchy=1"
```

Afterwards you have to execute `grub2-mkconfig -o /boot/grub2/grub.cfg` and reboot.

If you have a different distro, then the steps might be different.

> You might want to add some more parameters here, so read the next section first before you rewrite your boot loader and reboot.


### How To Enable Cgroup-based Accounting?

The simplest way would be to enable the accounting by creating an override file `/etc/systemd/system.conf.d/Accounting.conf`.
Leave the lines out, if you don't want to collect data about it. 

```
[Manager]
DefaultCPUAccounting=yes
DefaultIOAccounting=yes
DefaultIPAccounting=yes
DefaultBlockIOAccounting=yes
DefaultMemoryAccounting=yes
DefaultTasksAccounting=yes
```

Afterwards a reboot is in order. Some cgroup stuff cannot be changed after the hierarchy has been set up.


### Enabling Swap Accounting And Pressure Stall Information

If you find below `/sys/fs/cgroup/` a lot of `memory.*` files, but no `memory.swap.*` ones, then you have to enable swap accounting. Also check if there are pressure files (`{io,memory,cpu}.pressure`) and if they are readable! They might be
present, but if reading them leads to an `Operation not supported`, you still have to enable PSI.

To enable both, add `swapaccount=1 psi=1` to the kernel command line. How to do so, you can read in the previous section.


## The Parts Of `cgar`

The next sections just describe the components briefly. Where they have to be put and configured, will be explained later in the chapter 'Installation'.

### `cgar_collect`

The first step is to collect and log the cgroup data. This is the job of `cgar_collect`. \
It is meant to be called regularly and therefore has a very little footprint. It has been written in Go and simply reads the configured cgroup parameters and appends the data as a JSON object to a log file. Go routines parallelize this reading and leads to a short runtime which also benefits the accuracy.

> Sorry in case the golang code might be crappy. This was my first golang project and I learned the language along the way.

### `cgar_collector.service` And `cgar_collector.timer`

To collect the cgroup data periodically `cgar_collect` has to be called regularly. Nowadays `systemd` timers are the way to go. With each timer unit requiring a service unit, we end up with: cgar_collector.service` and `cgar_collector.timer`

### `cgar` (`logrotate`)

With `cgar_collect` always appending data, the log file inevitably grows and depending of the amount of data you collect, this can become quite a lot. To keep that in check, `logrotate` will to compress older data and delete obsolete data to save disk space.


## Supported Cgroup Data

Cgroup2 can account quite a lot of information. For now only parts of the memory controller are supported, to get the project running. It should be fairly easy to implement more parameters and controllers.

For now we only collect:

- `memory.current`
- `memory.high`
- `memory.min`
- `memory.max`
- `memory.low`
- `memory.stat`
- `memory.pressure`
- `memory.swap.current`
- `memory.swap.high`
- `memory.swap.max`

> *`cgar_log2csv` currently only knows about `memory.current`, `memory.swap.current` and `memory.pressure`!*


## Installation

1. Clone this repo in a directory.

2. Build and copy `cgar_collect`.

   ```
   cd cgar_collect
   go build
   ```

   Afterwards you should find the compiled binary `cgar_collect` in the same directory.

   You need obviously Go installed to compile it. If you don't want to do so, you find the binary already in the repo for the x86_64 architecture. 

3. Copy the binary to `/usr/local/sbin/`.

   ```
   cp cgar_collect /usr/local/sbin/
   cd ..
   ```

   The binary should already be executable, if not run `chmod +x /usr/local/sbin/cgar_collect`. \
   If you prefer another location, go ahead!

4. Setup the configuration and logging.

   ```
   mkdir /etc/cgar /var/log/cgar
   cp conf.json /etc/cgar 
   ```

   The shipped configuration will collect the data of the memory controller for all cgroups and tell `cgar_collect` to log it in `/var/log/cgar/cgar`. The chapter 'Configuration' below deals with the details how to change this.

5. Log rotation

   ```
   cp logrotate-cgar /etc/logrotate.d/cgar 
   ```

   The configuration will compress daily and keep the last 28 rotated logs.

6. Copy and enable the systemd units.

   ```
   cp cgar_collector.service cgar_collector.timer /etc/systemd/system/
   systemctl daemon-reload
   systemctl enable --now cgar_collector.timer
   ```

   If you have placed `cgar_collect` at a different location in the previous step, you have to adapt the path in `cgar_collector.service` before you enable the timer unit. 

   The timer unit will collect the cgroup data every minute. Adapt `cgar_collector.timer` if this is not what you want before you enable the timer unit. 

7. Check this everything works.

   If everything works well, the system log should record the minutely execution ogf `cgar_collect`:

   ```
   2023-02-05T16:21:05.239175+01:00 silver cgar_collect[11864]: Called as: /usr/local/sbin/cgar_collect
   2023-02-05T16:21:05.239242+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.current: no such file or directory
   2023-02-05T16:21:05.239279+01:00 silver cgar_collect[11864]:  100 [memory]
   2023-02-05T16:21:05.239311+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.high: no such file or directory
   2023-02-05T16:21:05.239338+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.min: no such file or directory
   2023-02-05T16:21:05.239368+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.low: no such file or directory
   2023-02-05T16:21:05.239536+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.swap.high: no such file or directory
   2023-02-05T16:21:05.239569+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.max: no such file or directory
   2023-02-05T16:21:05.239595+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.swap.current: no such file or directory
   2023-02-05T16:21:05.239626+01:00 silver cgar_collect[11864]: open /sys/fs/cgroup//memory.swap.max: no such file or directory
   2023-02-05T16:21:05.260144+01:00 silver cgar_collect[11864]: Terminated.
   ```

   *Don't worry about the "no such file or directory" lines. Some parent cgroups not necessarily have controller files by design. But in any case, investigate if there not really might be a misconfiguration.*

   Also the file `/var/log/cgar/cgar` should start to fill up:

   ```
   {"2023-02-05 14:16:05.237070196 +0100 CET m=+0.000711024":{"":{"memory.pressure":"some avg10=0.00 avg60=0.00 avg300=0.00 total=0\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0","memory.stat":"anon 3759386624\nfile 9145090048\nkernel_stack 21102592\npagetables 118468608\npercpu 10753968\nsock 61440\nshmem 1124151296\nfile_mapped 1145061376\nfile_dirty 2580480\nfile_writeback 0\nswapcached 0\nanon_thp 1002438656\nfile_thp 0\nshmem_thp 0\ninactive_anon...
   ```
   Don't be scared. Each line contains a JSON object with all the collect information. As everything gets collected as default, theses lines get really really long! Better use `jq`:

   ```
   jq < /var/log/cgar/cgar
   {
   "2023-02-05 16:19:01.595132787 +0100 CET m=+0.000459595": {
      "": {
         "memory.pressure": "some avg10=0.00 avg60=0.00 avg300=0.00 total=0\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0",
         "memory.stat": "anon 4411154432\nfile 9817915392\nkernel_stack  ..."
      },
      "dev-hugepages.mount": {
         "memory.current": "106496",
         "memory.high": "max",
         "memory.low": "0",
         "memory.max": "max",
         "memory.min": "0",
         "memory.pressure": "some avg10=0.00 avg60=0.00 avg300=0.00 total=0\nfull ... ",
         "memory.stat": "anon 0\nfile 102400\nkernel_stack 0\npagetables 0... ",
         "memory.swap.current": "0",
         "memory.swap.high": "max",
         "memory.swap.max": "max"
      },
   ...
   ```

## Configuration

   The configuration `/etc/cgar/conf.json` is a simple JSON file:

   ```
   {
    "logfile": "/var/log/cgar/cgar",     <-- this is the file where the collected data gets logged
    "collect": [                         
        {                                <-- each cgroup is an object '{..}' with three attributes
            "cgroup": "",                <-- name of the cgroup you want to (start to) collect 
            "depth": 100,                <-- how many children shall be traversed 
            "controllers": ["memory"]    <-- collect data for all those controllers
        }
      ...                                <-- put another cgroup object here
    ]
   }
   ```

## What To Do With All The Data?

For now we have a large log with JSON objects containing all the data we wanted. If you have something which can process JSON data, you already can go ahead. If not, the next step might be to get a readable table.

For this `cgar_log2csv` exists. It reads a log and prints it as CSV to stdout after applying filters to reduce the data:

```
usage: cgar_log2csv [-h] [--start-time TIME] [--end-time TIME] [-l LOGFILE]
                    [-f FILTER] [--unit-mem UNIT] [-s SEPARATOR] [-r]
                    [--quote-strings] [--group-by {cgroup,attribute}]
                    CGROUP [CGROUP ...]

Extracts cgroup data from cgar collector log and prints them as CSV on stdout.

positional arguments:
  CGROUP                extract data for this cgroup

optional arguments:
  -h, --help            show this help message and exit
  --start-time TIME     only print data from this time forward (no timezone
                        info)
  --end-time TIME       only print data to this time (no timezone info)
  -l LOGFILE, --log LOGFILE
                        logfile to parse (default: "/var/log/cgar/cgar")
  -f FILTER, --filter FILTER
                        print only these attributes
  --unit-mem UNIT       display memory in this unit (default: B(ytes))
  -s SEPARATOR, --separator SEPARATOR
                        separator for the output (default: ,
  -r, --regex           use regular expressions for CGROUP argument
  --quote-strings       if strings in CSV output shall be quoted (default:
                        false
  --group-by {cgroup,attribute}
                        how cgroup attribute columns shall be grouped
                        (default: cgroup

v0.3
```

Such a file can easily be read into a spreadsheet program like LibreOffice or Excel to draw nice graphs for example.
But you don't need an entire spreadsheet program to draw some graphs. This can easily be achieved with `gnuplot` as well.

For those who love large tables, the tool `cgar_csv2table` will read such a CSV file and print it as a nice human readable
table on stdout.

```
usage: cgar_csv2table [-h] [-s SEPARATOR] CSVFILE

Reads a CVS file created by `cgar_log2csv` and prints the data as table to
stdout.

positional arguments:
  CSVFILE               the CSV file

optional arguments:
  -h, --help            show this help message and exit
  -s SEPARATOR, --separator SEPARATOR
                        separator used in the CSV file (default: ,)

v0.1
```

Happy investigations!

## What To Do to Add New Cgroup Data?

First support for reading the controller files must be added to `cgar_collect`. \
Enhance the function  `ReadCgroupController()` (`cgroupcollect/cgroupcollect.go`), compile the project and
exchange the binary on the system.


## Ideas And ToDos

- Enhance the support for the memory controller.
- Implementing more cgroup controller.


## Changelog

   |||
   |-|-|
   | 06.02.2023 | First release on GitHub. `cgar_collect` works, `cgar_log2csv` needs a bit polishing. 
   | 06.02.2023 | `cgar_log2csv`: v0.2 released
   | 08.02.2023 | new versions: `cgar_collect` v0.1.1, `cgar_log2csv` v0.4.1 and `cgar_csv2table` v0.1
   | 09.02.2023 | `cgar_log2csv` v0.4.2 and `cgar_csv2table` v0.2
