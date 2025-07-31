FROM alpine:latest

# Create and set working directory
WORKDIR /app
# Copy our binary and configs to the container
COPY main ./
COPY configs/default.yaml ./
# Make sure the binary is executable
RUN chmod +x ./main
# Expose the port for accepting incoming requests
EXPOSE 8081
# Execute our service
CMD ["/app/main"]
