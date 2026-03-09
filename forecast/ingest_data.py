# ingest_data.py
import pandas as pd
from sqlalchemy import create_engine
import json


def load_metrics_df(limit: int = 10) -> pd.DataFrame:
    # Load database config
    with open('../config/db.config', 'r') as f:
        db_config = json.load(f)

    connection_string = (
        f"postgresql://{db_config['user']}:{db_config['password']}"
        f"@{db_config['host']}:{db_config['port']}/{db_config['dbname']}"
    )

    engine = create_engine(connection_string)

    query = f"""
        SELECT *
        FROM metrics
        WHERE value > 100
        LIMIT {limit};
    """

    df = pd.read_sql_query(query, con=engine)

    # Expand labels JSON
    df_expanded = pd.DataFrame(df["labels"].tolist())
    df = pd.concat([df.drop("labels", axis=1), df_expanded], axis=1)
    df = df.drop(columns=["__name__"], errors="ignore")

    engine.dispose()
    return df


if __name__ == "__main__":
    df = load_metrics_df()
    print(df.head())

