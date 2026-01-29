from fastapi import FastAPI
from dotenv import load_dotenv
import os

# Load environment variables
load_dotenv()

app = FastAPI(title="Mind Garden API")

@app.get("/")
def read_root():
    return {"status": "ok", "message": "Mind Garden API is running"}

@app.get("/healthz")
def health_check():
    return {"status": "healthy"}
