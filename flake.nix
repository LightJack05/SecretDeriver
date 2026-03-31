{
  description = "K8s Operator Dev Shell";

  inputs = {
    go-license-collector.url = "github:LightJack05/go-license-collector";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, go-license-collector, ... }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = [
          pkgs.kind
          pkgs.kubectl
          pkgs.operator-sdk
          go-license-collector.packages.${system}.go-license-collector
        ];

        KIND_EXPERIMENTAL_PROVIDER="podman";

        shellHook = ''
        export KUBECONFIG=$HOME/.kube/config.d/kind-six-nodes
        '';

      };
    };
}
