use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};

#[derive(Deserialize, Serialize, Debug)]
struct WarReport {
    country: String,
    warplanes_in_air: u32,
    warships_in_water: u32,
    timestamp: String,
}

#[post("/data")]
async fn receive_report(report: web::Json<WarReport>) -> impl Responder {
    println!("Reporte recibido: {:?}", report);
    
    // Aquí agregaremos posteriormente la lógica para enviar el dato a Go (gRPC Client)
    
    HttpResponse::Ok().json(serde_json::json!({
        "status": "success",
        "message": "Reporte militar procesado correctamente"
    }))
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