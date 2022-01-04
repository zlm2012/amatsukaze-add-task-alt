# Amatsukaze Add Task Alternative

(C) 2021 Lingmo Zhu

This is a simple utility as an alternative to [Amatsukaze](https://github.com/nekopanda/Amatsukaze) [Add Task](https://github.com/nekopanda/Amatsukaze/tree/master/AmatsukazeAddTask).

## Why

Though it is possible to use `AmatsukazeAddTask.exe` with [mono](https://www.mono-project.com/) on Linux, when the source file name includes space(the plain space with ASCII 0x20), the remaining part after the whitespace would be gone in the add queue request. I'm not sure if it is because mono or AmatsukazeAddTask.

And also, I think it's better to build a much more portable solution.

## Usage

`./amatsukaze-add-task -e </path/to/save/encoded/files/remote> -i </path/to/source> -p <profile name> [OPTIONS]`

Options

* `-e, --encode`: path to where to put the encoded files, from the remote (i.e. where Amatsukaze server runs) view
* `-i --input`: path to the input/source file (it could be from the local view, with `-r` option applied)
* `-r --remote`: path to the source file directory from the remote view
* `-c --connect`: the address of Amatsukaze server, with default value `127.0.0.1:32768`
* `-p --profile`: the name of the profile defined in Amatsukaze
* `-w --wol`: MAC address for wake on lan (Experimental)
* `-I --wol-iface`: local interface name to send the wake on lan packet (Experimental)

Example (please change the paths and parameters accordingly if you want to use this):

```bash
./amatsukaze-add-task -e '\\192.168.1.2\recorded\encoded' -r '\\192.168.1.2\recorded' -i '/mnt/recorded/test.m2ts' -w 'fe:ed:be:ef:de:ad' -I eth0 -c '192.168.1.3:32768' -p '自動選択_デフォルト'
```

## License

MIT License is applied. please check [LICENSE](./LICENSE) for detail.