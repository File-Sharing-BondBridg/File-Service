-- Create the database schema for file management
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    extension VARCHAR(10) NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL,
    file_path VARCHAR(500),
    preview_path VARCHAR(500),
    share_url VARCHAR(500),
    bucket_name VARCHAR(100) DEFAULT 'files',
    is_processed BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_type ON files(type);
CREATE INDEX IF NOT EXISTS idx_files_extension ON files(extension);

-- Create a function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE OR REPLACE TRIGGER update_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert some sample data for testing (optional)
INSERT INTO files (id, name, original_name, size, type, extension, uploaded_at, file_path, bucket_name) 
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'Sample Image', 'sample.jpg', 1024, 'image', '.jpg', NOW(), '11111111-1111-1111-1111-111111111111.jpg', 'files'),
    ('22222222-2222-2222-2222-222222222222', 'Sample Document', 'document.pdf', 2048, 'document', '.pdf', NOW(), '22222222-2222-2222-2222-222222222222.pdf', 'files')
ON CONFLICT (id) DO NOTHING;