# oslog-collector

# Motivation

macOS OS Log (also known as Unified logs) is not saved in text format on the file system, so some log aggregator/forwarder softwares may not support it.  
This software saves the OS Log obtained via the `log` command in a file in ndjson format so that it can be collected by other log collection tools.

# Install

```sh
$ brew tap mrtc0/tap
$ brew install mrtc0/tap/oslog-collector
```

# Configuration

When installed with Homebrew, the configuration file is placed in `/opt/homebrew/etc/oslog-collector.conf`.

```sh
$ cat /opt/homebrew/etc/oslog-collector.conf
pid_file: oslog-collector.pid
collectors:
  # example: collect com.apple.mdns logs
  - name: mdns
    predicate: "subsystem == 'com.apple.mdns'"
    output_file: /opt/homebrew/var/log/oslog-mdns.json
    position_file: /opt/homebrew/var/log/oslog-mds.pos
    interval: 30 # seconds
```

# Launch and Stop

```sh 
$ brew services start oslog-collector
$ brew services stop oslog-collector
```

oslog-collector logs are output to `(brew --prefix)/var/log/oslog-collector.log`.


