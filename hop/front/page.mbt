// Status page, agent card, and A2A replies for mazzy-front.
// The page never claims the egress is clean and never carries a URL.

pub fn field(text : String, key : String) -> String {
  let prefix = key + " "
  let mut rest = text
  while rest.length() > 0 {
    let i = match rest.find("\n") {
      Some(n) => n
      None => rest.length()
    }
    let line = rest.substring(end=i).replace(old="\r", new="")
    if i == rest.length() {
      rest = ""
    } else {
      rest = rest.substring(start=i + 1)
    }
    if line.has_prefix(prefix) {
      return line.substring(start=prefix.length())
    }
  }
  ""
}

fn count_exact(text : String, want : String) -> Int {
  let mut n = 0
  let mut rest = text
  while rest.length() > 0 {
    let i = match rest.find("\n") {
      Some(k) => k
      None => rest.length()
    }
    let line = rest.substring(end=i).replace(old="\r", new="")
    if line == want {
      n = n + 1
    }
    if i == rest.length() {
      break
    }
    rest = rest.substring(start=i + 1)
  }
  n
}

pub fn admin_addr(url : String) -> String {
  if !url.has_prefix("http://") {
    return ""
  }
  let rest = url.substring(start=7)
  if rest.length() == 0 ||
    rest.contains("/") ||
    rest.contains(" ") ||
    rest.contains("@") {
    return ""
  }
  match rest.split_once(":") {
    None => ""
    Some((host, port)) => {
      if host.length() == 0 || port.length() == 0 || port.contains(":") {
        return ""
      }
      let n = try Some(@string.parse_int(port.to_owned())) catch {
        _ => None
      }
      match n {
        Some(v) => if v >= 1 && v <= 65535 { rest } else { "" }
        None => ""
      }
    }
  }
}

pub fn program_ok(text : String) -> Bool {
  let bind = field(text, "a2a-bind")
  let cred = field(text, "admin-credential")
  bind.length() > 0 &&
  !bind.contains(" ") &&
  !bind.contains("REPLACE_") &&
  cred.length() >= 16 &&
  !cred.contains("REPLACE_") &&
  admin_addr(field(text, "admin-url")) != "" &&
  field(text, "transport") == "nested-https" &&
  field(text, "claim-clean") == "no" &&
  field(text, "require-auth") == "yes" &&
  count_exact(text, "artifact mazzy-core mcp") == 1 &&
  count_exact(text, "artifact mazzy-backend admin") == 1 &&
  count_exact(text, "artifact mazzy-front a2a") == 1 &&
  count_exact(text, "logic relay copy") == 1 &&
  count_exact(text, "logic protect mojo") == 1 &&
  count_exact(text, "logic doctor nim") == 1
}

pub fn bearer(headers : String) -> String {
  let mut rest = headers
  while rest.length() > 0 {
    let i = match rest.find("\n") {
      Some(n) => n
      None => rest.length()
    }
    let line = rest.substring(end=i).replace(old="\r", new="").trim().to_owned()
    if i == rest.length() {
      rest = ""
    } else {
      rest = rest.substring(start=i + 1)
    }
    if line.to_lower().has_prefix("authorization:") {
      let value = line.substring(start=14).trim().to_owned()
      if value.to_lower().has_prefix("bearer ") {
        return value.substring(start=7).trim().to_owned()
      }
    }
  }
  ""
}

pub fn escape_html(s : String) -> String {
  s.replace(old="&", new="&amp;").replace(old="<", new="&lt;").replace(old=">", new="&gt;")
}

pub fn page(doctor : String, sessions : String, protection : String) -> String {
  "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Mazzy hop</title></head><body><h1>Mazzy hop</h1><p>Nested HTTPS. The egress is not claimed clean.</p><pre>" +
  escape_html(doctor) +
  "</pre><pre>" +
  escape_html(sessions) +
  "</pre><pre>" +
  escape_html(protection) +
  "</pre></body></html>"
}

pub fn agent_card() -> String {
  "{\"name\":\"mazzy-front\",\"description\":\"Mazzy hop status. The egress is not claimed clean.\",\"version\":\"0.1.0\",\"capabilities\":{\"streaming\":false},\"skills\":[{\"id\":\"doctor\",\"name\":\"doctor\"},{\"id\":\"sessions\",\"name\":\"sessions\"},{\"id\":\"protection\",\"name\":\"protection\"}],\"defaultInputModes\":[\"text\"],\"defaultOutputModes\":[\"text\"]}"
}

pub fn json_escape(s : String) -> String {
  s.replace(old="\\", new="\\\\")
    .replace(old="\"", new="\\\"")
    .replace(old="\n", new="\\n")
    .replace(old="\r", new="\\r")
}

pub fn json_id(body : String) -> String {
  match body.find("\"id\":") {
    None => "null"
    Some(i) => {
      let rest = body.substring(start=i + 5).trim().to_owned()
      let mut out = ""
      let mut j = 0
      while j < rest.length() {
        let c = rest.substring(start=j, end=j + 1)
        if c == "," || c == "}" || c == " " || c == "\n" || c == "\r" {
          break
        }
        out = out + c
        j = j + 1
      }
      if out.length() == 0 {
        "null"
      } else {
        out
      }
    }
  }
}

pub fn skill_of(body : String) -> String {
  if body.contains("\"protection\"") {
    "protection"
  } else if body.contains("\"sessions\"") {
    "sessions"
  } else {
    "doctor"
  }
}

pub fn a2a_ok(id : String, text : String) -> String {
  "{\"jsonrpc\":\"2.0\",\"id\":" +
  id +
  ",\"result\":{\"kind\":\"message\",\"role\":\"agent\",\"parts\":[{\"kind\":\"text\",\"text\":\"" +
  json_escape(text) +
  "\"}]}}"
}

pub fn seal(body : String) -> String {
  let cleaned = body.replace(old="\"egress_clean\": true", new="\"egress_clean\": false")
    .replace(old="\"egress_clean\":true", new="\"egress_clean\":false")
  if cleaned.contains("egress_clean") {
    cleaned
  } else {
    "{\"egress_clean\":false,\"status\":\"unavailable\"}"
  }
}

pub fn http_response(status : String, kind : String, body : String) -> String {
  "HTTP/1.1 " +
  status +
  "\r\nContent-Type: " +
  kind +
  "\r\nContent-Length: " +
  body.length().to_string() +
  "\r\nConnection: close\r\n\r\n" +
  body
}

pub fn request_line(raw : String) -> (String, String) {
  let line = match raw.find("\r\n") {
    Some(i) => raw.substring(end=i)
    None => raw
  }
  match line.split_once(" ") {
    Some((verb, rest)) =>
      match rest.split_once(" ") {
        Some((path, _)) => (verb.to_owned(), path.to_owned())
        None => (verb.to_owned(), rest.to_owned())
      }
    None => ("", "")
  }
}

fn content_length(headers : String) -> Int32 {
  let mut rest = headers
  while rest.length() > 0 {
    let i = match rest.find("\n") {
      Some(n) => n
      None => rest.length()
    }
    let line = rest.substring(end=i).replace(old="\r", new="").trim().to_owned()
    if i == rest.length() {
      rest = ""
    } else {
      rest = rest.substring(start=i + 1)
    }
    if line.to_lower().has_prefix("content-length:") {
      let v = line.substring(start=15).trim().to_owned()
      let n = try Some(@string.parse_int(v)) catch {
        _ => None
      }
      return match n {
        Some(x) => if x > 0 && x <= 8192 { x } else { 0 }
        None => 0
      }
    }
  }
  0
}

pub fn route(
  verb : String,
  path : String,
  headers : String,
  body : String,
  credential : String,
  doctor : String,
  sessions : String,
  protection : String,
) -> String {
  if bearer(headers) != credential || credential.length() < 16 {
    return http_response("401 Unauthorized", "text/plain; charset=utf-8", "refused\n")
  }
  if verb == "GET" && (path == "/" || path == "") {
    return http_response(
      "200 OK",
      "text/html; charset=utf-8",
      page(doctor, sessions, protection),
    )
  }
  if verb == "GET" && path == "/.well-known/agent-card.json" {
    return http_response("200 OK", "application/json", agent_card())
  }
  if verb == "POST" && path == "/a2a" {
    let skill = skill_of(body)
    let text = if skill == "sessions" {
      sessions
    } else if skill == "protection" {
      protection
    } else {
      doctor
    }
    return http_response("200 OK", "application/json", a2a_ok(json_id(body), text))
  }
  http_response("404 Not Found", "text/plain; charset=utf-8", "refused\n")
}

pub fn read_body_length(headers : String) -> Int32 {
  content_length(headers)
}
