export SSHPASS:="myce"
export SSH_CMD := "sshpass -e ssh -t -oUserKnownHostsFile=/dev/null -oStrictHostKeyChecking=no myce@127.0.0.1 -p 4444"


# Choose the openfaas endpoint
OPENFAAS_PORT := env_var_or_default('OPENFAAS_PORT', "8080")
OPENFAAS := env_var_or_default('OPENFAAS', "http://127.0.0.1:" + OPENFAAS_PORT)

# Registry for storing images temporarily
REGISTRY := env_var_or_default('REGISTRY', "ttl.sh/" + `whoami`)

# Time that the image will be stored in the registry
TAG := env_var_or_default('TAG', "2h")

_default:
    @just --list

# Build the project as a docker image
container:
    nix build .#docker
    docker load < result

# connects inside the VM using SSH
ssh:
    @$SSH_CMD

faas-login namespace="openfaas":
    #!/usr/bin/env bash
    PASS=$($SSH_CMD sudo kubectl get secret -n {{namespace}} basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode)
    echo $PASS | $SSH_CMD faas-cli login -g {{OPENFAAS}} --password-stdin
    echo "PASSWORD: $PASS"

# Publish OpenFaaS functions using the registry variables and modify the function file
faas-pub:
    #!/usr/bin/env bash
    cd {{ justfile_directory() }}/functions
    for file in *.yml; do
        if [ -f "$file" ]; then
            just faas-pub-single $file
        fi
    done

# Publish a single OpenFaaS function using the registry variables and modify the function file
faas-pub-single file:
    #!/usr/bin/env bash
    cd {{ justfile_directory() }}/functions
    sed -i "s|image: .*|image: {{ REGISTRY }}/$(basename {{file}} .yml):{{ TAG }}|" "{{file}}"
    {{SSH_CMD}} << EOF
    cd /home/myce/mycelium/functions
    faas-cli publish -g {{OPENFAAS}} -f "{{file}}"
    faas-cli deploy -g {{OPENFAAS}} -f "{{file}}"
    EOF

mqtt-pub topic message:
    mosquitto_pub -h 127.0.0.1 -p 1883 -t "{{ topic }}" -m "{{ message }}"

mqtt-sub topic:
    mosquitto_sub -h 127.0.0.1 -p 1883 -t "{{ topic }}"

vm:
    #!/usr/bin/env bash
    set -xe
    vmpath=$(nix build --impure --print-out-paths --expr "
    let
    self = builtins.getFlake ''path://{{ justfile_directory() }}'';
      vm = self.nixosConfigurationsFunction.os {pwd=''{{ justfile_directory() }}'';};
      install = vm.config.system.build.vm;
    in
    install" )
    rm *.qcow2 || true
    exec $vmpath/bin/run-* -nographic -cpu host -enable-kvm

