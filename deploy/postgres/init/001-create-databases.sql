CREATE USER auth_app WITH PASSWORD 'auth_app_dev';
CREATE USER file_app WITH PASSWORD 'file_app_dev';
CREATE USER knowledge_app WITH PASSWORD 'knowledge_app_dev';
CREATE USER qa_app WITH PASSWORD 'qa_app_dev';
CREATE USER document_app WITH PASSWORD 'document_app_dev';
CREATE USER ai_gateway_app WITH PASSWORD 'ai_gateway_app_dev';

CREATE DATABASE auth_system OWNER auth_app;
CREATE DATABASE file_system OWNER file_app;
CREATE DATABASE knowledge_system OWNER knowledge_app;
CREATE DATABASE qa_system OWNER qa_app;
CREATE DATABASE document_system OWNER document_app;
CREATE DATABASE ai_gateway_system OWNER ai_gateway_app;
