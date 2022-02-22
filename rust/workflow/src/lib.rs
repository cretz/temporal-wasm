use serde::{Serialize, Serializer, Deserialize, Deserializer};
use std::collections::HashMap;

#[cfg(feature = "wee-alloc")]
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

mod host_calls {
    extern "C" {
        pub fn complete(output_offset: *const u8, output_len: usize);
        pub fn complete_with_failure(failure_offset: *const u8, failure_len: usize);
        pub fn get_info(info_offset: *mut u8, info_len: usize);
        pub fn get_info_len() -> usize;
    }
}

#[derive(Serialize, Deserialize, Default)]
pub struct Payload {
    #[serde(default, with = "base64_string_map", skip_serializing_if = "HashMap::is_empty")]
    metadata: HashMap<String, Vec<u8>>,
    #[serde(default, with = "base64", skip_serializing_if = "Vec::is_empty")]
    data: Vec<u8>,
}

#[derive(Deserialize)]
pub struct Info {
    #[serde(default)]
    params: Vec<Payload>,
}

impl Info {
    pub fn load() -> Self {
        let mut bytes = Vec::<u8>::with_capacity(unsafe { host_calls::get_info_len() });
        unsafe { host_calls::get_info(bytes.as_mut_ptr(), bytes.capacity()); }
        serde_json::from_slice(&bytes[..]).unwrap()
    }

    pub fn single_param<'de, D: Deserialize<'de>>(&self) -> Result<D, String> {
        todo!()
    }
}

#[derive(Serialize, Default)]
pub struct Failure {
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")] 
    r#type: Option<String>,
    #[serde(skip_serializing_if = "is_false")] 
    non_retryable: bool,
    #[serde(skip_serializing_if = "Vec::is_empty")] 
    details: Vec<Payload>,
    #[serde(skip_serializing_if = "Option::is_none")] 
    cause: Option<Box<Failure>>,
}

fn is_false(b: &bool) -> bool { !b }

mod base64 {
    use serde::{Serialize, Deserialize};
    use serde::{Deserializer, Serializer};

    pub fn serialize<S: Serializer>(v: &Vec<u8>, s: S) -> Result<S::Ok, S::Error> {
        let base64 = base64::encode(v);
        String::serialize(&base64, s)
    }
    
    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Vec<u8>, D::Error> {
        let base64 = String::deserialize(d)?;
        base64::decode(base64.as_bytes())
            .map_err(|e| serde::de::Error::custom(e))
    }
}

mod base64_string_map {
    use serde::{Serialize, Deserialize};
    use serde::{Deserializer, Serializer};
    use std::collections::HashMap;

    pub fn serialize<S: Serializer>(v: &HashMap<String, Vec<u8>>, s: S) -> Result<S::Ok, S::Error> {
        let new_map: HashMap<&String, String> = v.into_iter().map(|(k, v)| (k, base64::encode(v))).collect();
        HashMap::serialize(&new_map, s)
    }
    
    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<HashMap<String, Vec<u8>>, D::Error> {
        let new_map: HashMap<String, String> = HashMap::deserialize(d)?;
        new_map.
            into_iter().map(|(k, v)|
                base64::decode(v.as_bytes()).map(|v| (k, v)).map_err(|e| serde::de::Error::custom(e))
            ).collect()
    }
}