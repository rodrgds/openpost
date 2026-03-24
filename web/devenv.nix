{ pkgs, ... }:

{
  # Bun language support
  languages.javascript = {
    enable = true;
    bun.enable = true;
  };

  # Scripts for frontend development
  scripts = {
    web-dev.exec = ''
      cd web && bun install && bun run dev
    '';

    web-build.exec = ''
      cd web && bun install && bun run build
    '';

    web-test.exec = ''
      cd web && bun run test
    '';

    web-check.exec = ''
      cd web && bun run check
    '';

    web-lint.exec = ''
      cd web && bun run lint
    '';

    web-format.exec = ''
      cd web && bun run format
    '';
  };

  # Git hooks
  git-hooks.hooks = {
    prettier.enable = true;
    eslint.enable = true;
  };
}
