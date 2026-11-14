// mazzy-front owns A2A. It reads the toeac program through mazzy-toeac
// after mazzy-core has accepted the file. It does not relay tunnel bytes.

fn flag(args : Array[String], name : String, fallback : String) -> String {
  let mut i = 1
  while i < args.length() - 1 {
    if args[i] == name {
      return args[i + 1]
    }
    i = i + 1
  }
  fallback
}

async fn refuse() -> Unit {
  @stdio.stderr.write(@utf8.encode("front: refused\n"))
  @env.exit(78)
}

async fn load(
  config : String,
  trust : String,
  toeac : String,
) -> (String, String, String) {
  let trusted = @process.run(
    trust,
    ["trust-config", config],
    stdout=Discard,
    stderr=Discard,
    inherit_env=true,
  )
  if trusted.exit_code != 0 {
    refuse()
  }
  let parsed = @process.run(
    toeac,
    [config],
    stdout=Collect,
    stderr=Discard,
    inherit_env=true,
  )
  if parsed.exit_code != 0 {
    refuse()
  }
  let text = try parsed.stdout_text() catch {
    _ => ""
  }
  if !@front.program_ok(text) {
    refuse()
  }
  (
    @front.field(text, "a2a-bind"),
    @front.field(text, "admin-credential"),
    @front.field(text, "admin-url"),
  )
}

async fn main {
  let args = @env.args()
  let config = flag(args, "--config", "/etc/mazzy-vpn/hop.toeac")
  let trust = flag(args, "--trust", "mazzy-core")
  let toeac = flag(args, "--toeac", "mazzy-toeac")
  let (bind, cred, url) = load(config, trust, toeac)
  try {
    @front.serve(bind, cred, url)
  } catch {
    _ => refuse()
  }
}
