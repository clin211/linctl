version: "2"

linters:
  default: standard
  enable:
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - bodyclose
    - contextcheck
    - copyloopvar
    - dogsled
    - durationcheck
    - errorlint
    - forbidigo
    - gocritic
    - gofmt
    - goimports
    - misspell
    - noctx
    - prealloc
    - revive
    - unconvert

  disable:
    - exhaustruct
    - wsl
    - nlreturn
    - gci

linters-settings:
  goimports:
    local-prefixes: {{.Module}}
  revive:
    rules:
      - name: exported
        arguments:
          - disableStutteringCheck
  forbidigo:
    forbid:
      - pattern: '^fmt\.Print(f|ln)?$'
        message: "请使用 github.com/clin211/linhub/log 替代 fmt.Print*"

issues:
  exclude-rules:
    - path: "_test.go"
      linters:
        - errcheck
        - dupl
    - path: "wire_gen.go"
      linters:
        - all
