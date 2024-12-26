{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  dotenv.enable = true;
  difftastic.enable = true;

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.protoc-gen-go
    pkgs.protoc-gen-go
    pkgs.templ
    pkgs.go
    pkgs.gotools
    pkgs.govulncheck
    pkgs.gopls
    pkgs.golint
    pkgs.gofumpt
    pkgs.golangci-lint
    pkgs.gotestsum
    pkgs.gotestfmt
    pkgs.gotestdox
  ];

  # https://devenv.sh/languages/
  # languages.rust.enable = true;
  languages.go = {
    enable = true;
    enableHardeningWorkaround = true;
  };

  # https://devenv.sh/processes/
  # processes.cargo-watch.exec = "cargo-watch";

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/scripts/
  scripts.hello.exec = ''
    echo hello from $GREET
  '';

  enterShell = ''
    hello
    git --version
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    ${pkgs.gotestsum}/bin/gotestsum -f testdox --format-hide-empty-pkg -- -race ./...
  '';

  # https://devenv.sh/pre-commit-hooks/
  pre-commit.hooks = {
    shellcheck.enable = true;
    shfmt.enable = true;
    gofmt.enable = true;
    nixfmt-rfc-style.enable = true;
    golangci-lint = {
      enable = true;
      language = "golang";
      name = "golangci-lint";
      description = "Run golangci-lint on Go files";
      entry = "${pkgs.golangci-lint}/bin/golangci-lint run";
      types = [ "go" ];
      require_serial = true;
      pass_filenames = false;
    };
  };

  # See full reference at https://devenv.sh/reference/options/
}
