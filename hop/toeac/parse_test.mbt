fn sample() -> String {
  #|transport nested-https
  #|
  #|server {
  #|  bind "REPLACE_BIND"
  #|  cert "REPLACE_CERT_PATH"
  #|  key "REPLACE_KEY_PATH"
  #|}
  #|
  #|session {
  #|  idle 120s
  #|  max 32
  #|}
  #|
  #|doctor {
  #|  every 30s
  #|  claim-clean no
  #|}
  #|
  #|protect {
  #|  credential "REPLACE_HOP_CREDENTIAL"
  #|  allow "REPLACE_HOST:REPLACE_PORT"
  #|  require-auth yes
  #|}
  #|
  #|artifact {
  #|  process "mazzy-core"
  #|  owns mcp
  #|}
  #|
  #|artifact {
  #|  process "mazzy-backend"
  #|  owns admin
  #|}
  #|
  #|artifact {
  #|  process "mazzy-front"
  #|  owns a2a
  #|}
  #|
  #|logic {
  #|  relay copy
  #|  protect mojo
  #|  doctor nim
  #|}
  #|
  #|mcp {
  #|  socket "REPLACE_MCP_SOCKET"
  #|}
  #|
  #|admin {
  #|  bind "REPLACE_ADMIN_BIND"
  #|  credential "REPLACE_ADMIN_CREDENTIAL"
  #|  status "REPLACE_STATUS_SOCKET"
  #|}
  #|
  #|a2a {
  #|  bind "REPLACE_A2A_BIND"
  #|  admin "REPLACE_ADMIN_URL"
  #|}
}

test "placeholder program renders" {
  guard render(sample()) is Ok(text) else { fail("expected ok") }
  inspect(text.has_prefix("ok\n"), content="true")
  inspect(text.contains("transport nested-https"), content="true")
  inspect(text.contains("artifact mazzy-core mcp"), content="true")
  inspect(text.contains("artifact mazzy-backend admin"), content="true")
  inspect(text.contains("artifact mazzy-front a2a"), content="true")
  inspect(text.contains("logic relay copy"), content="true")
  inspect(text.contains("claim-clean no"), content="true")
  inspect(text.contains("egress_clean true"), content="false")
}

test "wireguard transport is refused" {
  let src = sample().replace(old="transport nested-https", new="transport wireguard")
  inspect(show_outcome(render(src)), content="refused")
}

test "claiming clean is refused" {
  let src = sample().replace(old="claim-clean no", new="claim-clean yes")
  inspect(show_outcome(render(src)), content="refused")
}

test "decrypt relay is refused" {
  let src = sample().replace(old="relay copy", new="relay decrypt")
  inspect(show_outcome(render(src)), content="refused")
}

test "wrong process for mcp is refused" {
  let src = sample().replace(old="process \"mazzy-core\"", new="process \"mazzy-front\"")
  inspect(show_outcome(render(src)), content="refused")
}

test "broken brace is bad syntax" {
  inspect(show_outcome(render("transport nested-https\n}\n")), content="bad")
}
