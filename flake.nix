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
              shellcheck.enable = true;
              shfmt.enable = true;
              gofmt.enable = true;
              nixfmt-rfc-style.enable = true;
              golangci-lint = {
                enable = true;
                name = "golangci-lint";
                description = "Run golangci-lint on Go files";
                entry = "${pkgs.golangci-lint}/bin/golangci-lint run";
                types = [ "go" ];
                pass_filenames = false;
              };
              actionlint = {
                enable = true;
                name = "actionlint";
                description = "Lint GitHub Actions workflow files";
                entry = "${pkgs.actionlint}/bin/actionlint";
                files = "^.github/workflows/.*\\.(yaml|yml)$";
                pass_filenames = false;
              };
            };
          };
        };

        devShell = pkgs.mkShell {
          packages = with pkgs; [
            actionlint
            git
            protoc-gen-go
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
            ${self.checks.${system}.pre-commit-check.shellHook}
          '';

          # Set GOROOT and other necessary Go environment variables
          GOROOT = "${pkgs.go}/share/go";
        };
      }
    );
}
