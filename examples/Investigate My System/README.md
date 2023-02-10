# Investigate My System

I wanted to know what consumes memory on my system.

After letting `cgar_collect` logging for a few hours, I made a copy of the growing log and started to create some CSV files and graphs.

```
# cp /var/log/cgar/cgar cgar.log
```

First an overview about the memory consumption:

```
# cgar_log2csv 'system.slice' 'user.slice' 'init.scope' -l cgar.log --quote-strings -f memory.current --unit-mem MiB > overview.csv
```

This command creates a CSV file `memory.csv` only with `memory.current` in MiB for the cgroups `system.slice`, `user.slice` and `init.scope`.
I got:

```
"timestamp","user.slice::memory.current","system.slice::memory.current","init.scope::memory.current"
"2023-02-09T08:59:00+01:00",0.0,1892.8,63.9
"2023-02-09T09:00:20+01:00",384.1,1642.6,64.3
"2023-02-09T09:01:21+01:00",384.0,1637.3,63.6
"2023-02-09T09:02:29+01:00",384.0,1633.4,63.7
...
```
 or a bit prettier:

```
# cgar_csv2table overview.csv
        timestamp           init.scope    system.slice    user.slice   
                          memory.current memory.current memory.current 
------------------------- -------------- -------------- --------------
2023-02-09T08:59:00+01:00           63.9         1892.8            0.0 
2023-02-09T09:00:20+01:00           64.3         1642.6          384.1 
2023-02-09T09:01:21+01:00           63.6         1637.3          384.0 
2023-02-09T09:02:29+01:00           63.7         1633.4          384.0 
...
```

> I know, that I left some processes out (`machine.slice`, etc.), but I did not expected a noticeable consumption there).

Time to make a graphic representation. Since I'm not yet familiar with `gnuplot` I just loaded the CSV into LibreOffice and plotted a simple graph:

![Graph with memory consumption for `system.slice`, `user.slice` and `init.scope`](overview.png "Overview Memory Consumption")

Clearly the memory hungry ones are in `user.slice`! \
System services seem to consume only a bit more then 2.5 GiB.

Nevertheless let's have look at them, just to know who the top consumers are.

```
# cgar_log2csv '^system\.slice/[^/]+$' -r  -l cgar.log --quote-strings -f memory.current --unit-mem MiB > system.slice.csv
```

This line is similar to the first one, but the cgroup part needs a bit explaining, I guess. The goal is to get only the direct children of `system.slice`, but not `system.slice` itself. With `-r` the usage of regular expressions is enabled and the regex for cgroup just matches everything starting with `system.slice/` (the trailling `/` prevents `system.slice` from being included), followed by only non-`/` characters (no subdirectories). 
 
I refrained myself from adding the CSV output, since each line is very long and not really readable. Let's jump directly to the graph:

![Graph with memory consumption for direct children of `system.slice`](system.slice.png "Memory Consumption: `system.slice`")

The top 5 are:

- `display-manager.service`: around 400 to 500 MiB
- `snap.wekan.wekan.service`: ~340 MiB
- `docker.service`: almost 300 MiB
- `snap.wekan.mongodb.service`: ~170 MiB
- `libvirtd.service`: ~160 MiB

Around 12:14 the ranking changes a bit and `cron.service` rises to #4 by allocating roughly 200 MiB suddendly.
I noticed this the day before. Same jump, same time. I assume a `cron.daily` run, but could not find any prove at first glance.

Besides the `cron` anomaly, clearly my Wekan installation has a great part in the 2.5 GiB.. :-) \

But we wanted to see, what is eating all the memory in the `user.slice`.
Let's have a closer look first, since the `user.slice` usually goes deeper then the `system.slice`:

```
# tree -d  /sys/fs/cgroup/user.slice/
/sys/fs/cgroup/user.slice/
└── user-99520.slice
    ├── session-3.scope
    └── user@99520.service
        ├── app.slice
        │   ├── app-\x2fusr\x2fbin\x2fkorgac-8fd079022f414a189c9618f2bf90e292.scope
        │   ├── app-\x2fusr\x2fbin\x2fspectacle-29600deb75934543a78ce66ffedf8fa4.scope
        ...
        │   ├── xdg-desktop-portal.service
        │   ├── xdg-document-portal.service
        │   └── xdg-permission-store.service
        ├── background.slice
        │   ├── plasma-baloorunner.service
        │   ├── plasma-kactivitymanagerd.service
        │   ├── plasma-kglobalaccel.service
        │   ├── plasma-krunner.service
        │   └── plasma-kscreen.service
        ├── init.scope
        └── session.slice
            ├── pipewire.service
            ├── plasma-xdg-desktop-portal-kde.service
            ├── pulseaudio.service
            └── wireplumber.service
```

Lets check first the consumption of `session-3.scope` and `user@99520.service`:

```
# cgar_log2csv '^user\.slice/user\-99520.slice/[^/]+$' -r -l cgar.log --quote-strings -f memory.current --unit-mem MiB > user-99520.slice.csv
user@99520.service
```

![Graph with memory consumption for direct children of `user-99520.slice`](user-99520.slice.png "Memory Consumption: `user-99520.slice`")

We have to take a closer look at `user@99520.service`:

```
# cgar_log2csv '^user\.slice/user\-99520.slice/user@99520.service/[^/]+$' -r -l cgar.log --quote-strings -f memory.current --unit-mem MiB > user@99520.service.csv
```

![Graph with memory consumption for direct children of `user@99520.service`](user@99520.service.png "Memory Consumption: `user@99520.service`")

Almost there! The `app.slice` clearly contains the processes with the highest consumption.

```
cgar_log2csv '^user\.slice/user\-99520.slice/user@99520.service/app.slice/[^/]+$' -r -l cgar.log --quote-strings -f memory.current --unit-mem MiB > app.slice.csv
```

![Graph with memory consumption for direct children of `app.slice`](app.slice.png "Memory Consumption: `app.slice`")

The top 5 are:

1. `app-firefox-56677d05c1114b53b946b301503537ea.scope` -> Firefox
2. `app-code-fda39f76d62f48cc98fe36303fd66b8b.scope` -> Visual Studio Code
3. `app-org.gnome.Evolution-2bc75e6dfb9147d69c1bb21f583faa12.scope` -> Evolution
4. `snap.slack.slack.cb3c1f22-38af-4b91-b980-355456fa3879.scope` -> Slack
5. `app-org.kde.konsole-40028973a9844b66a204587b06d23070.scope` -> KDE


But this is just the beginning. You can dig much deeper. Just as an idea some memory details for `Firefox`:

```
cgar_log2csv 'user.slice/user-99520.slice/user@99520.service/app.slice/app-firefox-56677d05c1114b53b946b301503537ea.scope' -l cgar.log --quote-strings -f memory.stat::anon -f memory.stat::file -f memory.stat::kernel_stack -f memory.stat::pagetables -f memory.stat::percpu -f memory.stat::sock -f memory.stat::shmem --unit-mem MiB > firefox_details.csv
```

![Graph with memory details for `app-firefox-56677d05c1114b53b946b301503537ea.scope`](firefox_details.png "Memory Details: `app-firefox-56677d05c1114b53b946b301503537ea.scope`")

Here the documentation for the details of `memory.stayt`: https://docs.kernel.org/admin-guide/cgroup-v2.html#memory-interface-files

Have fun!