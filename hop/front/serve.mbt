// mazzy-front speaks ordinary HTTP. It calls the admin API in cleartext
// on the address from the toeac program, so it never skips a TLS check.
// MCP stays with mazzy-core. The admin API stays with mazzy-backend.

async fn ask(admin_url : String, path : String, credential : String) -> String {
  try {
    let addr = admin_addr(admin_url)
    if addr == "" {
      return "{\"egress_clean\":false,\"status\":\"unavailable\"}"
    }
    let conn = @socket.Tcp::connect(@socket.Addr::parse(addr))
    defer conn.close()
    let req =
      "GET " +
      path +
      " HTTP/1.1\r\nHost: " +
      addr +
      "\r\nAuthorization: Bearer " +
      credential +
      "\r\nConnection: close\r\n\r\n"
    conn.write(@utf8.encode(req))
    let raw = @utf8.decode_lossy(conn.read_all())
    let body = match raw.find("\r\n\r\n") {
      Some(i) => raw.substring(start=i + 4)
      None => ""
    }
    seal(body)
  } catch {
    _ => "{\"egress_clean\":false,\"status\":\"unavailable\"}"
  }
}

async fn handle(conn : @socket.Tcp, credential : String, admin_url : String) -> Unit {
  defer conn.close()
  try {
    let head = conn.read_until(@utf8.encode("\r\n\r\n"), include_delimiter=true)
    match head {
      None => ()
      Some(raw) => {
        let text = @utf8.decode_lossy(raw)
        let (verb, path) = request_line(text)
        if bearer(text) != credential || credential.length() < 16 {
          conn.write(
            @utf8.encode(
              http_response("401 Unauthorized", "text/plain; charset=utf-8", "refused\n"),
            ),
          )
          return
        }
        let n = read_body_length(text)
        let mut body = ""
        if n > 0 {
          body = @utf8.decode_lossy(conn.read_exactly(n))
        }
        let reply = if verb == "GET" && path == "/.well-known/agent-card.json" {
          route(verb, path, text, body, credential, "", "", "")
        } else if verb == "GET" && (path == "/" || path == "") {
          let doctor = ask(admin_url, "/v1/doctor", credential)
          let sessions = ask(admin_url, "/v1/sessions", credential)
          let protection = ask(admin_url, "/v1/protection", credential)
          route(verb, path, text, body, credential, doctor, sessions, protection)
        } else if verb == "POST" && path == "/a2a" {
          let skill = skill_of(body)
          let api = if skill == "sessions" {
            "/v1/sessions"
          } else if skill == "protection" {
            "/v1/protection"
          } else {
            "/v1/doctor"
          }
          let report = ask(admin_url, api, credential)
          route(verb, path, text, body, credential, report, report, report)
        } else {
          route(verb, path, text, body, credential, "", "", "")
        }
        conn.write(@utf8.encode(reply))
      }
    }
  } catch {
    _ => ()
  }
}

pub async fn serve(bind : String, credential : String, admin_url : String) -> Unit {
  let server = @socket.TcpServer(@socket.Addr::parse(bind), reuse_addr=true)
  server.run_forever() <| ((conn, _) => {
    handle(conn, credential, admin_url)
  })
}
