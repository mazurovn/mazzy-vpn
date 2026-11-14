test "page does not claim a clean egress" {
  let html = page("a<b", "sessions", "protection")
  inspect(html.contains("a&lt;b"), content="true")
  inspect(html.contains("not claimed clean"), content="true")
}

test "anonymous status is refused" {
  let raw = route("GET", "/", "", "", "admin-credential-ok", "doctor-report", "s", "p")
  inspect(raw.has_prefix("HTTP/1.1 401"), content="true")
  inspect(raw.contains("doctor-report"), content="false")
}

test "agent card has no address" {
  let card = agent_card()
  inspect(card.contains("mazzy-front"), content="true")
  inspect(card.contains("http"), content="false")
}

test "admin url stays on clear http" {
  inspect(admin_addr("http://127.0.0.1:80"), content="127.0.0.1:80")
  inspect(admin_addr("https://127.0.0.1:80"), content="")
  inspect(admin_addr("http://127.0.0.1:80/v1"), content="")
}

test "a2a selects the sessions skill" {
  let raw = route(
    "POST",
    "/a2a",
    "Authorization: Bearer admin-credential-ok\r\n",
    "{\"id\":7,\"text\":\"sessions\"}",
    "admin-credential-ok",
    "doctor-text",
    "session-text",
    "protection-text",
  )
  inspect(raw.contains("session-text"), content="true")
  inspect(raw.contains("\"id\":7"), content="true")
  inspect(raw.contains("doctor-text"), content="false")
}

test "seal keeps egress dirty" {
  inspect(seal("{\"egress_clean\":true,\"sessions\":1}").contains("\"egress_clean\":false"), content="true")
  inspect(seal("nope").contains("unavailable"), content="true")
}

test "program refuses a foreign owner" {
  let text =
    "transport nested-https\nclaim-clean no\nrequire-auth yes\na2a-bind 127.0.0.1:9\nadmin-credential admin-credential-ok\nadmin-url http://127.0.0.1:8\nartifact mazzy-core mcp\nartifact mazzy-backend admin\nlogic relay copy\nlogic protect mojo\nlogic doctor nim\n"
  inspect(program_ok(text), content="false")
}
