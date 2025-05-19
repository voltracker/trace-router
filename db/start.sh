docker build . -t trace-router-db
# obviously going to change the pw if i was actually deploying this
docker run --name trdb -e POSTGRES_PASSWORD=testing -d trace-router-db
