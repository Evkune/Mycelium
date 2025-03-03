# Definitions can be imported from a separate file like this one
{
  self,
  lib,
  inputs,
  ...
}: let
  inherit (inputs) kubenix openfaas;
in {
  flake = {
    nixosModules.kubernetes = {
      pkgs,
      kubenix,
      ...
    }: {
      imports = [kubenix.modules.k8s kubenix.modules.helm];
      kubernetes.helm.releases = {
        openfaas = {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "openfaas";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/openfaas/ $out
            '';
          };
        };
        mqtt-connector = {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "sample-topic";
            clientID = "m1";
          };
        };
        mqtt-rawdata = {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "rawData";
            clientID = "m2";
          };
        };
        mqtt-analyseddata = {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "analysedData";
            clientID = "m3";
          };
        };
      mqtt-triggeranalyse= {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "triggerAnalyse";
            clientID = "m4";
          };
        };
      mqtt-triggeralert= {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "triggerAlert";
            clientID = "m5";
          };
        };
      mqtt-secondtopic= {
          namespace = lib.mkForce "openfaas";
          overrideNamespace = false;
          chart = pkgs.stdenvNoCC.mkDerivation {
            name = "mqtt-connector";
            src = openfaas;

            buildCommand = ''
              ls $src
              cp -r $src/chart/mqtt-connector/ $out
            '';
          };
          values = {
            broker = "tcp://10.0.2.15:1883";
            topic = "second-topic";
            clientID = "m5";
          };
        };
      };
      kubernetes.resources.deployments = {
        gateway.spec.template.spec.containers.gateway.image = lib.mkForce "ghcr.io/volodiapg/openfaas/gateway:0.27.2";
        gateway.spec.template.spec.containers.faas-netes.image = lib.mkForce "ghcr.io/volodiapg/openfaas/faas-netes:0.17.1";
        queue-worker.spec.template.spec.containers.queue-worker.image = lib.mkForce "ghcr.io/volodiapg/openfaas/queue-worker:0.14.0";
      };
    };

    nixosModules.kube = {pkgs, ...}: {
      programs.bash.shellAliases = {
        kubectl = "k3s kubectl";
        k = "sudo kubectl";
        k9 = "sudo k9s --kubeconfig /etc/rancher/k3s/k3s.yaml -A";
      };

      systemd.services.startTutoSEDContainer = {
        description = "Launch our tutosed container image";
        after = ["k3s.service"];
        wants = ["k3s.service"];
        wantedBy = ["multi-user.target"];
        script = ''
          ${pkgs.k3s}/bin/k3s kubectl apply -f /etc/namespaces.yaml
          ${pkgs.k3s}/bin/k3s kubectl apply -f /etc/kubenix.json
        '';
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = "yes";
        };
      };
      # Service de configuration pour InfluxDB
      systemd.services.influxdb2-init = {
        description = "Configuration initiale d'InfluxDB";
        wants = [ "influxdb2.service" ];        # Assure que `influxdb` est démarré avant
        after = [ "influxdb2.service" ];         # Exécute après le démarrage d'InfluxDB
        serviceConfig = {
          ExecStart = let
            influxSettings = {
              http-bind-address = "0.0.0.0:8086";
              auth-enabled = false;
              log-enabled = false;
              write-tracing = false;
              pprof-enabled = false;
              https-enabled = false;
            };
            script = pkgs.writeScript "influxdb2-init" ''
              #!${pkgs.runtimeShell}
              until ${pkgs.curl}/bin/curl -s -f -o /dev/null "http://${toString influxSettings.http-bind-address}"
              do
                sleep 5
              done

              # Configuration initiale d'InfluxDB
              ${pkgs.influxdb2-cli}/bin/influx setup \
                --host http://${toString influxSettings.http-bind-address} \
                --username admin \
                --password adminfaasfog \
                --token "Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==" \
                --org Mycelium \
                --bucket Bucket1 \
                --force

                export INFLUX_TOKEN=Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==


              # Création des autres buckets avec tokens spécifiques
              ${pkgs.influxdb2-cli}/bin/influx bucket create --host http://${toString influxSettings.http-bind-address} --org Mycelium --name Bucket2
              ${pkgs.influxdb2-cli}/bin/influx bucket create --host http://${toString influxSettings.http-bind-address} --org Mycelium --name FloodMonitoring
            '';
          in "${script} %u";
          Type = "oneshot";   # Le service s'exécute une fois puis s'arrête
        };
        wantedBy = ["multi-user.target"]; # S'assure que le service s'exécute au démarrage
      };
      services = {
        k3s = {
          enable = true;
        };

        # Active le service InfluxDB
        influxdb2 = {
          enable = true;
          package = pkgs.influxdb2-server;
        };
        mosquitto = {
          enable = true;

          listeners = [
            {
              address = "0.0.0.0";
              port = 1883;
              settings.allow_anonymous = true;
              omitPasswordAuth = true;
              acl = ["topic readwrite #" "pattern readwrite #"];
            }
          ];
        };
      };

      environment = {
        systemPackages = with pkgs; [
          k9s
          mosquitto
          influxdb
        ];
        etc = {
          "kubenix.json".source =
            (kubenix.evalModules.${pkgs.system} {
              module = self.outputs.nixosModules.kubernetes;
            })
            .config
            .kubernetes
            .result;
          "namespaces.yaml".text = ''
            apiVersion: v1
            kind: Namespace
            metadata:
              name: openfaas
              annotations:
                linkerd.io/inject: enabled
                config.linkerd.io/skip-inbound-ports: "4222"
                config.linkerd.io/skip-outbound-ports: "4222"
              labels:
                role: openfaas-system
                access: openfaas-system
                istio-injection: enabled
            ---
            apiVersion: v1
            kind: Namespace
            metadata:
              name: openfaas-fn
              annotations:
                linkerd.io/inject: enabled
                config.linkerd.io/skip-inbound-ports: "4222"
                config.linkerd.io/skip-outbound-ports: "4222"
              labels:
                istio-injection: enabled
                role: openfaas-fn
          '';
        };
      };
      system.stateVersion = "22.05"; # Do not change
    };
  };
}
