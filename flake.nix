{
  description = "Bubo Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks = {
      url = "github:cachix/pre-commit-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      pre-commit-hooks,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        checks = {
          # Add the pre-commit hooks. Some are not supported by default
          # so we add them manually.
          pre-commit-check = pre-commit-hooks.lib.${system}.run {
            src = ./.;
            hooks = {
              actionlint = {
                enable = true;
                name = "actionlint";
                description = "Lint GitHub Actions workflow files";
                entry = "${pkgs.actionlint}/bin/actionlint";
                files = "^.github/workflows/.*\\.(yaml|yml)$";
                pass_filenames = false;
              };
              markdownlint.enable = true;
              nixfmt-rfc-style.enable = true;
              gofumpt = {
                enable = true;
                name = "gofumpt";
                description = "Run gofumpt on Go files";
                entry = "${pkgs.gofumpt}/bin/gofumpt -l -w";
                files = "\\.go$";
                pass_filenames = true;
              };
              golangci-lint = {
                enable = true;
                name = "golangci-lint";
                description = "Run golangci-lint on Go files";
                entry = "env CGO_ENABLED=1 GOROOT=${pkgs.go}/share/go PATH=${pkgs.go}/bin:$PATH ${pkgs.golangci-lint}/bin/golangci-lint run";
                types = [ "go" ];
                pass_filenames = false;
              };
              shellcheck.enable = true;
              shfmt.enable = true;
            };
          };
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            actionlint
            gcc
            pkg-config
            git
            go
            golint
            gofumpt
            golangci-lint
            gopls
            goreleaser
            gotestsum
            gotestfmt
            gotestdox
            gotools
            govulncheck
          ];

          shellHook = ''
            go mod tidy
            ${self.checks.${system}.pre-commit-check.shellHook}
          '';

        };
      }
    );
}
