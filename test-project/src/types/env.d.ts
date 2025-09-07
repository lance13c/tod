declare namespace NodeJS {
  interface ProcessEnv {
    DATABASE_URL: string;
    BETTER_AUTH_SECRET: string;
    BETTER_AUTH_URL: string;
    NEXT_PUBLIC_APP_URL: string;
    
    // SMTP Configuration
    SMTP_HOST?: string;
    SMTP_PORT?: string;
    SMTP_SECURE?: string;
    SMTP_USER?: string;
    SMTP_PASS?: string;
    FROM_EMAIL?: string;
    FROM_NAME?: string;
    
    // OpenCage Geocoding API
    OPENCAGE_API_KEY?: string;
    
    // DuckDB Configuration
    DUCKDB_READ_ONLY?: string;
  }
}