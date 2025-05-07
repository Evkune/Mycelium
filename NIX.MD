# Introduction to Nix

This demo focuses on producing docker images and VMs to illustrate the main steps towards a full scale deployment on Grid'5000.

## Getting started

This tutorial works best on Linux, especially for the VM part that needs a linux x86 system to be generated.

1. Install nix: https://nixos.org/download#download-nix
2. Append to /etc/nix/nix.conf:
    ```
    extra-experimental-features = nix-command flakes
    max-jobs = auto
    cores = 0
    log-lines = 50
    ```
3. Now, from the root of this repository you can `nix develop` to enter the dev shell
4. Run `just` will list all available commands
7. `just vm` will boot the VM locally
8. `just ssh` will connect you inside
10. See it running on k3s with `k9` (`just ssh` and then `k9`)

Enjoy

# How to use Openfaas ?

1. `just vm`
3. `just faas-login`
4. Push a function with `just faas-pub` (consider changing the registry from what I set here. By default, the `REGISTRY` env var directs to ttl.sh, a temporary public registry)
5. You may deploy a single function with `just faas-pub-single <name of the yml file>`
6. `just mqtt-sub sample-topic` to subscribe to a mqtt topic (`sample-topic`)
6. `just mqtt-pub sample-topic "Hello World!"` to public on an mqtt topic (`sample-topic`)
