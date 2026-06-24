# rivr demo — deterministic & offline (stub backend), so the GIF never hangs on a network.
# Regenerate: `vhs demo/rivr.tape` (needs rivr on PATH: `go install ./cmd/rivr`).
Output demo/rivr.gif
Output demo/_verify.txt

Set Shell bash
Set FontSize 16
Set Width 1180
Set Height 700
Set Padding 20
Set Theme "Catppuccin Mocha"

Hide
Type "export PATH=$HOME/.local/share/mise/installs/go/1.25.5/bin:$PATH" Enter
Type "export RIVR_PROVIDER=stub NO_COLOR=1; clear" Enter
Show

Type "# rivr — read-only Amazon Shopping for agents. Bounded JSON, fenced text." Enter Sleep 1s
Type "rivr search 'usb-c cable' --json | jq '.items[0] | {asin,title,price,url}'" Enter Sleep 2.5s

Type "# untrusted text is fenced; deep links carry a disclosed affiliate tag" Enter Sleep 800ms
Type "rivr item get B0CXYZ123 --json --select title,url" Enter Sleep 2.5s

Type "# one normalized schema across backends — pick by capability/cost/risk" Enter Sleep 800ms
Type "rivr provider list --json --select name,keyless,hostedSafe,reviewsScope" Enter Sleep 3s

Type "# structured, stable exit codes + read-only safety state" Enter Sleep 800ms
Type "rivr schema | jq '{read_only: .safety.read_only, exit_codes}'" Enter Sleep 3s
