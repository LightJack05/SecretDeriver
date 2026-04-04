{
  description = "K8s Operator Dev Shell";

  inputs = {
    go-license-collector.url = "github:LightJack05/go-license-collector";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, go-license-collector, ... }:
    let
      helmify = pkgs.buildGoModule rec {
        pname = "helmify";
        version = "0.4.19";

        src = pkgs.fetchFromGitHub {
          owner = "arttor";
          repo = "helmify";
          rev = "v${version}";
          hash = "sha256-casZJRHpTbakI1FdSYPYcda2G8xPP+feTQil6R+FdbE=";
        };
        vendorHash = "sha256-ShX11hDA8oPkvT37vYArL8VzthfWdxbq9mvpYuL/0gE=";

        subPackages = [ "cmd/helmify" ];
      };

      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = [
          pkgs.kind
          pkgs.kubectl
          pkgs.operator-sdk
          pkgs.kustomize
          helmify
          go-license-collector.packages.${system}.go-license-collector
        ];

        KIND_EXPERIMENTAL_PROVIDER="podman";

        shellHook = ''
        export KUBECONFIG=$HOME/.kube/config.d/kind-six-nodes
        '';

      };
    };
}
