use temporal_wasm::{LogLevel, Info, Payload, complete, write_log};
use std::collections::HashMap;


// TODO(cretz): Make a macro that accepts arg and result signature and function
// body and automatically does param conversion, completion, etc
#[no_mangle]
pub fn run() {
    write_log(LogLevel::Info, "Workflow started");

    let info = Info::load();
    write_log(LogLevel::Info, &format!("Param count: {}", info.params.len()));

    assert!(info.params.len() == 1, "expected single param");
    let param_payload = info.params.first().unwrap();
    assert!(param_payload.metadata.get("encoding") == Some(&b"json/plain".to_vec()), "bad param encoding");
    let param: String = serde_json::from_slice(&param_payload.data).unwrap();

    let result = format!("Hello, {}!", param);
    let result_payload = Payload {
        metadata: HashMap::from([("encoding".to_owned(), b"json/plain".to_vec())]),
        data: serde_json::to_vec(&result).unwrap(),
    };
    complete(Some(result_payload));
}