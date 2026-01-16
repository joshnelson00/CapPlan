import pandas as pd
from sqlalchemy import create_engine
import json

# Load database config
with open('../config/db.config', 'r') as f:
    db_config = json.load(f)

# Create SQLAlchemy connection string
connection_string = f"postgresql://{db_config['user']}:{db_config['password']}@{db_config['host']}:{db_config['port']}/{db_config['dbname']}"

# Create SQLAlchemy engine
engine = create_engine(connection_string)

# Query data
query = ''' SELECT * 
            FROM metrics 
            WHERE value > 100 LIMIT 10; '''

df = pd.read_sql_query(query, con=engine)
# Clean/Format Data
df_expanded = pd.DataFrame(df['labels'].tolist())
df = pd.concat([df.drop('labels', axis=1), df_expanded], axis=1)
df = df.drop(columns = ['__name__'])

print(df)


# Close the connection
engine.dispose()
