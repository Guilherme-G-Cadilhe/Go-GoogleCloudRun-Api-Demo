# Estágio de construção
FROM golang:1.24-alpine AS builder

# Define o diretório de trabalho dentro do contêiner
WORKDIR /app

# Copia os arquivos go.mod e go.sum para gerenciar as dependências
COPY go.mod ./
# COPY go.sum ./

# Baixa as dependências do Go
RUN go mod download

# Copia o código-fonte da aplicação
COPY . .

# Constrói a aplicação Go
# CGO_ENABLED=0 é para criar um binário estaticamente linkado, o que é bom para imagens menores
# -o weather-app define o nome do executável
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o weather-app cmd/main.go

# Estágio final (imagem menor para produção)
FROM alpine:latest

# Define o diretório de trabalho
WORKDIR /root/

# Copia o executável do estágio de construção
COPY --from=builder /app/weather-app .

# Expõe a porta que a aplicação irá escutar
EXPOSE 8080

# Define a variável de ambiente para a porta (padrão para Cloud Run)
ENV PORT=8080

# Comando para executar a aplicação
CMD ["./weather-app"]

