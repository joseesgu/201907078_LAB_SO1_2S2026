use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use reqwest::Client;

#[derive(Deserialize, Serialize, Debug, Clone)]
struct WarReport {
    country: String,
    warplanes_in_air: u32,
    warships_in_water: u32,
    timestamp: String,
}

#[post("/data")]
async fn receive_report(report: web::Json<WarReport>) -> impl Responder {
    let client = Client::new();
    // Apunta al servicio de K8s del Deployment de Go
    let go_url = std::env::var("GO_CLIENT_URL").unwrap_or_else(|_| "http://go-client-svc:8081/send".to_string());

    let payload = report.into_inner();
    
    // Envía el dato al cliente Go
    match client.post(&go_url).json(&payload).send().await {
        Ok(_) => HttpResponse::Ok().json(serde_json::json!({
            "status": "success",
            "message": "Enviado a Go correctamente"
        })),
        Err(e) => {
            eprintln!("Error enviando a Go: {}", e);
            HttpResponse::InternalServerError().json(serde_json::json!({"status": "error"}))
        }
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("Servidor Rust escuchando en el puerto 8080...");
    HttpServer::new(|| {
        App::new().service(receive_report)
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}