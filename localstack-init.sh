#!/bin/bash

# Инициализация LocalStack для Report Service

set -e

echo "Initializing LocalStack S3 for Report Service..."

# Создаем S3 bucket для отчетов
awslocal s3 mb s3://report-srv-bucket --region us-east-1

# Устанавливаем CORS политику для bucket
awslocal s3api put-bucket-cors --bucket report-srv-bucket --cors-configuration '{
  "CORSRules": [
    {
      "AllowedOrigins": ["*"],
      "AllowedMethods": ["GET", "PUT", "POST", "DELETE"],
      "AllowedHeaders": ["*"],
      "MaxAgeSeconds": 3000
    }
  ]
}'

# Создаем папку для отчетов
awslocal s3api put-object --bucket report-srv-bucket --key reports/ --content-length 0

# Создаем папку для шаблонов
awslocal s3api put-object --bucket report-srv-bucket --key templates/ --content-length 0

# Проверяем что bucket создан
awslocal s3 ls

echo "LocalStack S3 initialized successfully!"
echo "Bucket: report-srv-bucket"
echo "Endpoint: http://localstack:4566" 