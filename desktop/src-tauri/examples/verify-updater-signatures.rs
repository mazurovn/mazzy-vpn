// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

use base64::{Engine as _, engine::general_purpose::STANDARD};
use minisign_verify::{PublicKey, Signature};
use serde_json::Value;
use std::{env, fs, io::Read, path::Path};

fn fail(message: impl std::fmt::Display) -> ! {
    eprintln!("updater-signature-audit: {message}");
    std::process::exit(1);
}

fn bounded_text(path: &Path, limit: u64) -> String {
    let metadata = fs::metadata(path)
        .unwrap_or_else(|error| fail(format!("cannot stat {}: {error}", path.display())));
    if !metadata.is_file() || metadata.len() == 0 || metadata.len() > limit {
        fail(format!("invalid bounded file: {}", path.display()));
    }
    fs::read_to_string(path)
        .unwrap_or_else(|error| fail(format!("cannot read {}: {error}", path.display())))
}

fn main() {
    let args: Vec<_> = env::args_os().skip(1).collect();
    if args.len() < 3 || args.len() % 2 == 0 {
        fail("usage: verify-updater-signatures TAURI_CONFIG ARTIFACT SIGNATURE [...]");
    }

    let config_path = Path::new(&args[0]);
    let config: Value = serde_json::from_str(&bounded_text(config_path, 1024 * 1024))
        .unwrap_or_else(|error| fail(format!("invalid Tauri config: {error}")));
    let encoded_key = config
        .pointer("/plugins/updater/pubkey")
        .and_then(Value::as_str)
        .filter(|value| value.len() <= 4096)
        .unwrap_or_else(|| fail("Tauri updater public key is missing"));
    let decoded_key = STANDARD
        .decode(encoded_key)
        .unwrap_or_else(|error| fail(format!("invalid updater public-key base64: {error}")));
    let decoded_key = String::from_utf8(decoded_key)
        .unwrap_or_else(|error| fail(format!("updater public key is not UTF-8: {error}")));
    let public_key = PublicKey::decode(&decoded_key)
        .unwrap_or_else(|error| fail(format!("invalid updater public key: {error}")));

    for pair in args[1..].chunks_exact(2) {
        let artifact_path = Path::new(&pair[0]);
        let signature_path = Path::new(&pair[1]);
        let encoded_signature = bounded_text(signature_path, 8192);
        let decoded_signature = STANDARD
            .decode(encoded_signature.trim())
            .unwrap_or_else(|error| {
                fail(format!(
                    "invalid signature base64 for {}: {error}",
                    artifact_path.display()
                ))
            });
        let decoded_signature = String::from_utf8(decoded_signature).unwrap_or_else(|error| {
            fail(format!(
                "signature for {} is not UTF-8: {error}",
                artifact_path.display()
            ))
        });
        let signature = Signature::decode(&decoded_signature).unwrap_or_else(|error| {
            fail(format!(
                "invalid signature for {}: {error}",
                artifact_path.display()
            ))
        });
        let mut verifier = public_key
            .verify_stream(&signature)
            .unwrap_or_else(|error| {
                fail(format!(
                    "signature key mismatch for {}: {error}",
                    artifact_path.display()
                ))
            });
        let mut artifact = fs::File::open(artifact_path).unwrap_or_else(|error| {
            fail(format!("cannot open {}: {error}", artifact_path.display()))
        });
        let mut buffer = [0_u8; 64 * 1024];
        loop {
            let read = artifact.read(&mut buffer).unwrap_or_else(|error| {
                fail(format!("cannot read {}: {error}", artifact_path.display()))
            });
            if read == 0 {
                break;
            }
            verifier.update(&buffer[..read]);
        }
        verifier.finalize().unwrap_or_else(|error| {
            fail(format!(
                "signature verification failed for {}: {error}",
                artifact_path.display()
            ))
        });
        println!(
            "updater-signature-audit: verified {}",
            artifact_path.display()
        );
    }
}
