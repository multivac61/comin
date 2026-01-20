{ self }: { config, pkgs, lib, ... }:
let
  cfg = config;
  cominConfigLib = import ./comin-config.nix { inherit config pkgs lib; };
  inherit (cominConfigLib) cominConfig cominConfigYaml;

  inherit (pkgs.stdenv.hostPlatform) system;
  inherit (cfg.services.comin) package;
in {
  imports = [ ./module-options.nix ];
  config = lib.mkIf cfg.services.comin.enable {
    assertions = [
      { assertion = package != null; message = "`services.comin.package` cannot be null."; }
      { assertion = package == null -> lib.elem system (lib.attrNames self.packages); message = "comin: ${system} is not supported by the Flake."; }
    ];

    environment.systemPackages = [ package ];
    services.comin.package = lib.mkDefault pkgs.comin or self.packages.${system}.comin or null;
    launchd.daemons.comin = {
      command = lib.concatStringsSep " " ([
        (lib.getExe package)
      ] ++ (lib.optionals cfg.services.comin.debug [ "--debug" ]) ++ [
        "run"
        "--config"
        "${cominConfigYaml}"
      ]);
      serviceConfig = {
        Label = "com.github.nlewo.comin";
        KeepAlive = true;
        RunAtLoad = true;
        StandardErrorPath = "/var/log/comin.log";
        StandardOutPath = "/var/log/comin.log";
        EnvironmentVariables = {
          PATH = lib.makeBinPath [ config.nix.package pkgs.git pkgs.openssh ];
        };
      };
    };

    system.activationScripts.comin.text = ''
      mkdir -p /var/lib/comin
      chown root:wheel /var/lib/comin
      chmod 755 /var/lib/comin
    '';

    # Comin manages its own restart through the deployment process.
    # This activation script ensures the service is running after activation,
    # handling cases where launchd may not have started it automatically.
    system.activationScripts.extraActivation.text = lib.mkAfter ''
      if [ -f /Library/LaunchDaemons/com.github.nlewo.comin.plist ]; then
        # Check if service is loaded (launchctl print succeeds if loaded)
        if /bin/launchctl print system/com.github.nlewo.comin &>/dev/null; then
          # Service is loaded, check if it has a PID (is running)
          if ! /bin/launchctl print system/com.github.nlewo.comin 2>/dev/null | grep -q "pid = [0-9]"; then
            echo "comin: service loaded but not running, starting..."
            /bin/launchctl kickstart system/com.github.nlewo.comin || true
          fi
        else
          # Service not loaded, load it (launchd will start it due to RunAtLoad)
          echo "comin: loading service..."
          /bin/launchctl load -w /Library/LaunchDaemons/com.github.nlewo.comin.plist || true
        fi
      fi
    '';
  };
}
