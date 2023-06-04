# Kube Config Merge Tool

This is a tool which you can use to merge kubernetes access config files to one main file.

## Usage

The command bollow allow you to recursevely merge all files from directoryes `~/configs/k8s/old`, `/var/my-work/k8s-configs`
and from file `/etc/k8s/admin` and from file `~/.kube/config` and save the result to destination `~/.kube/config`.

By default, this tool works in add mode. This means that all content from the source files will be added
to the target (`--target`) file, so the content of the target file will not be removed or modyfied.

```bash
kube-config-merger --directory ~/configs/k8s/old --directory /var/my-work/k8s-configs --file /etc/k8s/admin --target ~/.kube/config
```

## Command tool flags

| Flag name     | Default value    | Description |
|:-------------:|:----------------:|-------------|
| `--target`    | `~/.kube/config` | The path to the file to merge all the contents there.
| `--directory` |                  | The directory where configs will be searched. This configs will be merged. You can specify many directories.
| `--file`      |                  | The path to concrete config file which will be merged. You can specify many files.
| `-v`          |                  | Set this flag to show more details.
