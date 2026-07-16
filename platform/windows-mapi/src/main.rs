#[cfg(windows)]
fn main() {
    // Intentionally read-only scaffold. Add a signed COM registration layer and
    // subscribe to documented MAPI notifications only after tenant approval.
    println!("SCP-2317 MAPI provider scaffold");
}
#[cfg(not(windows))]
fn main() { eprintln!("Windows only"); }
