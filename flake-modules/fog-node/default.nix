{
  inputs,
  lib,
  ...
}: {
  imports = [
    ./kube.nix
  ];

  flake = {cfg, ...}: let
    modules = [
      "${inputs.nixpkgs}/nixos/modules/profiles/all-hardware.nix"
      inputs.self.outputs.nixosModules.base
      inputs.self.outputs.nixosModules.kube
      inputs.srvos.nixosModules.server
      inputs.srvos.nixosModules.mixins-terminfo
      inputs.srvos.nixosModules.mixins-systemd-boot
      inputs.srvos.nixosModules.mixins-nix-experimental
      ({pkgs, ...}: {
        boot.kernelPackages = pkgs.linuxPackages_latest;
        systemd.network = {
          enable = true;
          wait-online.anyInterface = true;
          networks = {
            "10-dhcp" = {
              matchConfig.Name = ["enp*" "wlp*"];
              networkConfig.DHCP = true;
            };
          };
        };
        networking.firewall.allowedUDPPorts = [67];
        services.getty.autologinUser =
          cfg.user;
        users.users.${cfg.user} = {
          isNormalUser = true;
          home = "/home/${cfg.user}";
          extraGroups = ["wheel" "networkmanager" "docker"]; # Add the user to important groups
          password = "myce";
        };
        security.sudo.wheelNeedsPassword = false;
        # Enable a basic firewall (optional)
        networking.firewall.enable = lib.mkForce false;
      })
    ];
  in {
    nixosConfigurations.base = inputs.nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      inherit modules;
      specialArgs = {inherit (inputs.self) outputs;};
    };

    nixosConfigurationsFunction.os = {pwd}:
      inputs.nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules =
          modules
          ++ [
            "${inputs.nixpkgs}/nixos/modules/profiles/qemu-guest.nix"
            ({pkgs, ...}: {
              environment.systemPackages = with pkgs; [
                just
                faas-cli
              ];
              virtualisation.docker.enable = true;
              virtualisation.vmVariant.virtualisation = {
                forwardPorts = [
                  # SSH
                  {
                    from = "host";
                    host.port = 4444;
                    guest.port = 22;
                  }
                  # MQTT
                  {
                    from = "host";
                    host.port = 1883;
                    guest.port = 1883;
                  }
                  # InfluxDB
                  {
                    from = "host";
                    host.port = 8086;
                    guest.port = 8086;
                  }
                  # OpenFaaS
                  {
                    from = "host";
                    host.port = 8080;
                    guest.port = 8080;
                  }
                  # OpenFaaS 2
                  {
                    from = "host";
                    host.port = 8082;
                    guest.port = 8082;
                  }
                  # Mosquitto 2
                  {
                    from = "host";
                    host.port = 1884;
                    guest.port = 1884;
                  }
                ];
                memorySize = 4096;
                cores = 4;
                diskSize = 10 * 1024;
                sharedDirectories.current = {
                  source = "${pwd}";
                  target = "/home/${cfg.user}/mycelium";
                };
              };
            })
          ];
        specialArgs = {inherit (inputs.self) outputs;};
      };

    nixosModules.base = {lib, ...}: {
      fileSystems."/" = {
        device = "/dev/disk/by-label/nixos";
        fsType = "ext4";
      };

      boot = {
        growPartition = true;
        kernelParams = ["console=ttyS0"]; # "preempt=none"];
        loader.grub = {
          device = "/dev/vda";
          configurationLimit = 5;
        };
        kernel.sysctl = {
          "net.core.default_qdisc" = lib.mkForce "cake"; #fq_codel also works but is older, allows for fair bandwidth for each application running on this node
          "net.ipv4.tcp_ecn" = 1;
          "net.ipv4.tcp_sack" = 1;
          "net.ipv4.tcp_dsack" = 1;
        };
      };

      users.mutableUsers = false;
      services = {
        openssh = {
          enable = true;
          passwordAuthentication = lib.mkForce true;
        };
      };

      system.stateVersion = "22.05"; # Do not change
    };
  };
}

