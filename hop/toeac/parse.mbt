// toeac configures the nested HTTPS hop: server, session, doctor,
// protection, which process owns MCP, A2A, and the admin API, and the
// logic each engine is allowed to run. String values are copied into the
// result and are never included in an error.

pub enum Outcome {
  Ok(String)
  Bad
  Refused
}

struct St {
  mut transport : String
  mut bind : String
  mut cert : String
  mut key : String
  mut idle : Int
  mut maxn : Int
  mut doctor : Int
  mut claim : String
  mut credential : String
  mut require_auth : String
  allows : Array[String]
  mut mcp_socket : String
  mut admin_bind : String
  mut admin_credential : String
  mut status_socket : String
  mut a2a_bind : String
  mut admin_url : String
  mut saw_core : Bool
  mut saw_backend : Bool
  mut saw_front : Bool
  mut relay : String
  mut protect : String
  mut doctor_engine : String
  mut block : String
  mut saw_idle : Bool
  mut saw_max : Bool
  mut saw_doctor : Bool
  mut err : String
}

fn fresh() -> St {
  {
    transport: "",
    bind: "",
    cert: "",
    key: "",
    idle: 0,
    maxn: 0,
    doctor: 0,
    claim: "",
    credential: "",
    require_auth: "",
    allows: [],
    mcp_socket: "",
    admin_bind: "",
    admin_credential: "",
    status_socket: "",
    a2a_bind: "",
    admin_url: "",
    saw_core: false,
    saw_backend: false,
    saw_front: false,
    relay: "",
    protect: "",
    doctor_engine: "",
    block: "",
    saw_idle: false,
    saw_max: false,
    saw_doctor: false,
    err: "",
  }
}

fn refuse(st : St) -> Unit {
  if st.err == "" {
    st.err = "refused"
  }
}

fn bad(st : St) -> Unit {
  if st.err == "" {
    st.err = "bad"
  }
}

fn once(st : St, cur : String, value : String) -> String {
  if cur != "" {
    bad(st)
    cur
  } else {
    value
  }
}

fn char_at(s : String, i : Int) -> Char? {
  s.get_char(i)
}

fn read_quoted(line : String) -> String? {
  let s = line.trim().to_owned()
  if !s.has_prefix("\"") {
    return None
  }
  let mut out = ""
  let mut i = 1
  while i < s.length() {
    let c = match char_at(s, i) {
      Some(ch) => ch
      None => return None
    }
    if c == '"' {
      let rest = s.substring(start=i + 1).trim()
      if rest.length() != 0 {
        return None
      }
      return Some(out)
    }
    if c == '\\' {
      i = i + 1
      let n = match char_at(s, i) {
        Some(ch) => ch
        None => return None
      }
      if n == '\\' {
        out = out + "\\"
      } else if n == '"' {
        out = out + "\""
      } else {
        return None
      }
    } else {
      out = out + c.to_string()
    }
    i = i + 1
  }
  None
}

fn all_digits(s : String) -> Bool {
  if s.length() == 0 {
    return false
  }
  let mut i = 0
  while i < s.length() {
    match char_at(s, i) {
      Some(c) =>
        if c < '0' || c > '9' {
          return false
        }
      None => return false
    }
    i = i + 1
  }
  true
}

fn seconds_of(v : String) -> Int? {
  let t = v.trim().to_owned()
  let n = if t.has_suffix("s") && t.length() > 1 {
    t.substring(end=t.length() - 1)
  } else {
    t
  }
  if !all_digits(n) {
    return None
  }
  try Some(@string.parse_int(n)) catch {
    _ => None
  }
}

fn in_range(n : Int, lo : Int, hi : Int) -> Bool {
  n >= lo && n <= hi
}

fn placeholder(s : String) -> Bool {
  s.contains("REPLACE_")
}

fn host_ok(host : String) -> Bool {
  if host.length() == 0 || host.length() > 253 {
    return false
  }
  let mut i = 0
  while i < host.length() {
    match char_at(host, i) {
      Some(c) =>
        if !(
          (c >= 'a' && c <= 'z') ||
          (c >= 'A' && c <= 'Z') ||
          (c >= '0' && c <= '9') ||
          c == '.' ||
          c == '-' ||
          c == '_',
        ) {
          return false
        }
      None => return false
    }
    i = i + 1
  }
  true
}

fn allow_ok(v : String) -> Bool {
  if placeholder(v) {
    return v.length() > 0 && not(v.contains(" ")) && not(v.contains("/"))
  }
  let cut = v.split_once(":")
  match cut {
    Some((host, port)) => {
      if host.contains(":") || !host_ok(host.to_owned()) {
        return false
      }
      let port_s = port.to_owned()
      if !all_digits(port_s) {
        return false
      }
      let n = try Some(@string.parse_int(port_s)) catch {
        _ => None
      }
      match n {
        Some(v) => in_range(v, 1, 65535)
        None => false
      }
    }
    None => false
  }
}

fn path_ok(v : String) -> Bool {
  if placeholder(v) {
    return true
  }
  v.has_prefix("/") && !v.contains("..") && !v.contains(" ")
}

fn banned(word : String) -> Bool {
  let w = word.to_lower()
  w == "amnezia" ||
  w == "amneziawg" ||
  w == "wireguard" ||
  w == "awg" ||
  w == "wg"
}

fn set_block(st : St, name : String) -> Unit {
  if st.block != "" {
    bad(st)
  }
  if name == "server" ||
    name == "session" ||
    name == "doctor" ||
    name == "protect" ||
    name == "artifact" ||
    name == "logic" ||
    name == "mcp" ||
    name == "admin" ||
    name == "a2a" {
    st.block = name
  } else {
    bad(st)
  }
}

fn take_word(st : St, line : String) -> Unit {
  let t = line.trim().to_owned()
  let cut = t.split_once(" ")
  let key = match cut {
    Some((k, _)) => k.to_string()
    None => t
  }
  let rest = match cut {
    Some((_, v)) => v.trim().to_owned()
    None => ""
  }
  if key == "" || banned(key) {
    bad(st)
    return
  }
  if st.block == "" {
    bad(st)
    return
  }
  if st.block == "server" {
    let quoted = match read_quoted(rest) {
      Some(s) => s
      None => {
        bad(st)
        return
      }
    }
    if quoted.length() == 0 || quoted.contains("\n") {
      bad(st)
      return
    }
    if key == "bind" {
      st.bind = once(st, st.bind, quoted)
    } else if key == "cert" {
      st.cert = once(st, st.cert, quoted)
    } else if key == "key" {
      st.key = once(st, st.key, quoted)
    } else {
      bad(st)
    }
  } else if st.block == "session" {
    let n = match seconds_of(rest) {
      Some(v) => v
      None => {
        bad(st)
        return
      }
    }
    if key == "idle" {
      if st.saw_idle {
        bad(st)
      }
      st.saw_idle = true
      if not(in_range(n, 1, 86400)) {
        refuse(st)
      }
      st.idle = n
    } else if key == "max" {
      if st.saw_max {
        bad(st)
      }
      st.saw_max = true
      if not(in_range(n, 1, 4096)) {
        refuse(st)
      }
      st.maxn = n
    } else {
      bad(st)
    }
  } else if st.block == "doctor" {
    if key == "every" {
      if st.saw_doctor {
        bad(st)
      }
      st.saw_doctor = true
      let n = match seconds_of(rest) {
        Some(v) => v
        None => {
          bad(st)
          return
        }
      }
      if not(in_range(n, 1, 86400)) {
        refuse(st)
      }
      st.doctor = n
    } else if key == "claim-clean" {
      st.claim = once(st, st.claim, rest)
      if rest != "no" {
        refuse(st)
      }
    } else {
      bad(st)
    }
  } else if st.block == "protect" {
    if key == "credential" {
      let quoted = match read_quoted(rest) {
        Some(s) => s
        None => {
          bad(st)
          return
        }
      }
      if quoted.length() < 16 || quoted.length() > 256 {
        refuse(st)
      }
      st.credential = once(st, st.credential, quoted)
    } else if key == "allow" {
      let quoted = match read_quoted(rest) {
        Some(s) => s
        None => {
          bad(st)
          return
        }
      }
      if not(allow_ok(quoted)) {
        bad(st)
        return
      }
      let mut i = 0
      while i < st.allows.length() {
        if st.allows[i] == quoted {
          bad(st)
          return
        }
        i = i + 1
      }
      st.allows.push(quoted)
    } else if key == "require-auth" {
      st.require_auth = once(st, st.require_auth, rest)
      if rest != "yes" {
        refuse(st)
      }
    } else {
      bad(st)
    }
  } else if st.block == "artifact" {
    if key == "process" {
      let quoted = match read_quoted(rest) {
        Some(s) => s
        None => {
          bad(st)
          return
        }
      }
      if banned(quoted) {
        refuse(st)
        return
      }
      st.block = "artifact:" + quoted
    } else {
      bad(st)
    }
  } else if st.block.has_prefix("artifact:") {
    let proc_name = st.block.substring(start=9)
    if key == "owns" {
      if rest == "mcp" && proc_name == "mazzy-core" {
        if st.saw_core {
          bad(st)
        }
        st.saw_core = true
      } else if rest == "admin" && proc_name == "mazzy-backend" {
        if st.saw_backend {
          bad(st)
        }
        st.saw_backend = true
      } else if rest == "a2a" && proc_name == "mazzy-front" {
        if st.saw_front {
          bad(st)
        }
        st.saw_front = true
      } else {
        refuse(st)
      }
    } else {
      bad(st)
    }
  } else if st.block == "logic" {
    if key == "relay" {
      st.relay = once(st, st.relay, rest)
      if rest != "copy" {
        refuse(st)
      }
    } else if key == "protect" {
      st.protect = once(st, st.protect, rest)
      if rest != "mojo" {
        refuse(st)
      }
    } else if key == "doctor" {
      st.doctor_engine = once(st, st.doctor_engine, rest)
      if rest != "nim" {
        refuse(st)
      }
    } else {
      bad(st)
    }
  } else if st.block == "mcp" {
    if key == "socket" {
      let quoted = match read_quoted(rest) {
        Some(s) => s
        None => {
          bad(st)
          return
        }
      }
      if not(path_ok(quoted)) {
        bad(st)
        return
      }
      st.mcp_socket = once(st, st.mcp_socket, quoted)
    } else {
      bad(st)
    }
  } else if st.block == "admin" {
    let quoted = match read_quoted(rest) {
      Some(s) => s
      None => {
        bad(st)
        return
      }
    }
    if key == "bind" {
      if quoted.length() == 0 {
        bad(st)
      }
      st.admin_bind = once(st, st.admin_bind, quoted)
    } else if key == "credential" {
      if quoted.length() < 16 || quoted.length() > 256 {
        refuse(st)
      }
      st.admin_credential = once(st, st.admin_credential, quoted)
    } else if key == "status" {
      if not(path_ok(quoted)) {
        bad(st)
        return
      }
      st.status_socket = once(st, st.status_socket, quoted)
    } else {
      bad(st)
    }
  } else if st.block == "a2a" {
    let quoted = match read_quoted(rest) {
      Some(s) => s
      None => {
        bad(st)
        return
      }
    }
    if key == "bind" {
      if quoted.length() == 0 {
        bad(st)
      }
      st.a2a_bind = once(st, st.a2a_bind, quoted)
    } else if key == "admin" {
      if quoted.length() == 0 || quoted.contains(" ") {
        bad(st)
      }
      st.admin_url = once(st, st.admin_url, quoted)
    } else {
      bad(st)
    }
  } else {
    bad(st)
  }
}

fn finish(st : St) -> Outcome {
  if st.err == "bad" {
    return Bad
  }
  if st.err == "refused" {
    return Refused
  }
  if st.block != "" && not(st.block.has_prefix("artifact:")) {
    return Bad
  }
  if st.transport != "nested-https" {
    return Refused
  }
  if st.bind == "" ||
    st.cert == "" ||
    st.key == "" ||
    not(st.saw_idle) ||
    not(st.saw_max) ||
    not(st.saw_doctor) ||
    st.claim != "no" ||
    st.credential == "" ||
    st.require_auth != "yes" ||
    st.allows.length() == 0 ||
    st.mcp_socket == "" ||
    st.admin_bind == "" ||
    st.admin_credential == "" ||
    st.status_socket == "" ||
    st.a2a_bind == "" ||
    st.admin_url == "" ||
    not(st.saw_core) ||
    not(st.saw_backend) ||
    not(st.saw_front) ||
    st.relay != "copy" ||
    st.protect != "mojo" ||
    st.doctor_engine != "nim" {
    return Refused
  }
  let mut out = "ok\n"
  out = out + "transport nested-https\n"
  out = out + "bind " + st.bind + "\n"
  out = out + "cert " + st.cert + "\n"
  out = out + "key " + st.key + "\n"
  out = out + "idle " + st.idle.to_string() + "\n"
  out = out + "max " + st.maxn.to_string() + "\n"
  out = out + "doctor " + st.doctor.to_string() + "\n"
  out = out + "claim-clean no\n"
  out = out + "require-auth yes\n"
  out = out + "credential " + st.credential + "\n"
  let mut i = 0
  while i < st.allows.length() {
    out = out + "allow " + st.allows[i] + "\n"
    i = i + 1
  }
  out = out + "mcp-socket " + st.mcp_socket + "\n"
  out = out + "admin-bind " + st.admin_bind + "\n"
  out = out + "admin-credential " + st.admin_credential + "\n"
  out = out + "status-socket " + st.status_socket + "\n"
  out = out + "a2a-bind " + st.a2a_bind + "\n"
  out = out + "admin-url " + st.admin_url + "\n"
  out = out + "artifact mazzy-core mcp\n"
  out = out + "artifact mazzy-backend admin\n"
  out = out + "artifact mazzy-front a2a\n"
  out = out + "logic relay copy\n"
  out = out + "logic protect mojo\n"
  out = out + "logic doctor nim\n"
  Ok(out)
}

fn handle_line(st : St, raw : String) -> Unit {
  if st.err != "" {
    return
  }
  let line = raw.trim().to_owned()
  if line.length() == 0 || line.has_prefix("#") {
    return
  }
  if banned(line) {
    refuse(st)
    return
  }
  if line == "}" {
    if st.block == "" {
      bad(st)
    } else if st.block.has_prefix("artifact:") {
      if not(
        st.block == "artifact:mazzy-core" && st.saw_core ||
        st.block == "artifact:mazzy-backend" && st.saw_backend ||
        st.block == "artifact:mazzy-front" && st.saw_front,
      ) {
        // The process was named but owns was missing or mismatched.
        if st.block == "artifact:mazzy-core" ||
          st.block == "artifact:mazzy-backend" ||
          st.block == "artifact:mazzy-front" {
          refuse(st)
        } else {
          refuse(st)
        }
      }
      st.block = ""
    } else {
      st.block = ""
    }
    return
  }
  if line.has_suffix("{") {
    let name = line.substring(end=line.length() - 1).trim().to_owned()
    if name.contains(" ") || name.length() == 0 {
      bad(st)
      return
    }
    set_block(st, name)
    return
  }
  if line == "transport nested-https" && st.block == "" {
    st.transport = once(st, st.transport, "nested-https")
    return
  }
  if line.has_prefix("transport ") && st.block == "" {
    st.transport = once(st, st.transport, "other")
    refuse(st)
    return
  }
  take_word(st, line)
}

pub fn render(src : String) -> Outcome {
  let st = fresh()
  let mut rest = src
  while true {
    match rest.find("\n") {
      Some(i) => {
        handle_line(st, rest.substring(end=i).replace(old="\r", new=""))
        rest = rest.substring(start=i + 1)
      }
      None => {
        if rest.length() != 0 {
          handle_line(st, rest.replace(old="\r", new=""))
        }
        break
      }
    }
  }
  finish(st)
}

pub fn show_outcome(o : Outcome) -> String {
  match o {
    Ok(s) => "ok\n" + s
    Bad => "bad"
    Refused => "refused"
  }
}
