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
        packages = {
          #   bubo-tool-gen = pkgs.buildGoModule {
          #     pname = "bubo-tool-gen";
          #     version = "0.1.0";
          #     src = ./.;
          #     subPackages = [
          #       "./cmd/bubo-tool-gen"
          #       "./executor"
          #       "./executor/pubsub"
          #       "./provider"
          #     ];
          #     # subPackages = [ "." ];
          #     # subPackages = [ "cmd/bubo-tool-gen" ];

          #     modRoot = ".";

          #     # This will be updated with the correct hash after first build
          #     vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

          #     env = {
          #       CGO_ENABLED = "0";
          #       GO111MODULE = "on";
          #       GOFLAGS = "-modfile=go.mod";
          #     };

          #     buildPhase = ''
          #       runHook preBuild
          #       cd $modRoot
          #       go build -v -o $out/bin/bubo-tool-gen ./cmd/bubo-tool-gen
          #       runHook postBuild
          #     '';
          #   };

          #   default = self.packages.${system}.bubo-tool-gen;
        };

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
                entry = "env PATH=${pkgs.go}/bin:${pkgs.diffutils}/bin:$PATH ${pkgs.golangci-lint}/bin/golangci-lint run";
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
            diffutils
            gcc
            pkg-config
            git
            go
            go-mockery
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
            just
            temporal
            # self.packages.${system}.bubo-tool-gen
          ];

          shellHook = ''
            go mod tidy
            ${self.checks.${system}.pre-commit-check.shellHook}
          '';

        };

      }
    );
}
