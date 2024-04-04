# fiwatcherd

## File Integrity Watcher Daemon

A simple lightweight service to watch one or more files for changes, and either alert or take action when they do.

It simply polls the filesystem to do this, keeping state on each watched file in memory.

This tool is not ready to be used by anyone right now, it was made to solve a simple domain-specific problem with a single file that contained an integer value.

In the future I will add the functionality listed in the TODOs to generalize this tool into something more useful.

