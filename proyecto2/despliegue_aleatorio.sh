#!/bin/bash

# Arreglo con los nombres de las imágenes locales
IMAGENES=("imagen-alto-ram" "imagen-alto-cpu" "imagen-bajo-consumo" "imagen-intruso")

echo "Iniciando el despliegue de 5 contenedores aleatorios..."

# Ciclo para orquestar la creación de exactamente 5 contenedores
for i in {1..5}; do
    # Obtener un índice aleatorio basado en la cantidad de imágenes disponibles
    INDICE_ALEATORIO=$((RANDOM % ${#IMAGENES[@]}))
    IMAGEN_SELECCIONADA=${IMAGENES[$INDICE_ALEATORIO]}
    
    # Generar un nombre único usando el timestamp para evitar conflictos de nombres
    NOMBRE_CONTENEDOR="contenedor_${IMAGEN_SELECCIONADA}_$(date +%s)_${i}"
    
    echo "Lanzando contenedor $i: $IMAGEN_SELECCIONADA"
    
    # Ejecutar el contenedor en segundo plano (detached mode)
    docker run -d --name "$NOMBRE_CONTENEDOR" "$IMAGEN_SELECCIONADA"
done

echo "Despliegue finalizado."