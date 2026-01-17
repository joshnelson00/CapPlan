import pandas as pd
from typing import List, Tuple
import joblib

import ingest_data

from sklearn.model_selection import train_test_split
from sklearn.preprocessing import OneHotEncoder
from sklearn.compose import ColumnTransformer
from sklearn.pipeline import Pipeline
from sklearn.metrics import mean_absolute_error, mean_squared_error

from xgboost import XGBRegressor


# ----------------------------
# Configuration
# ----------------------------

TARGET_COLUMN: str = "value_rate"  # We'll create this from counter diffs
TIMESTAMP_COLUMN: str = "timestamp"

TEST_SIZE: float = 0.2
RANDOM_STATE: int = 42


# ----------------------------
# Feature Engineering
# ----------------------------

def add_time_features(df: pd.DataFrame, ts_col: str) -> pd.DataFrame:
    """
    Expand timestamp into ML-friendly time features.
    """
    df = df.copy()
    df[ts_col] = pd.to_datetime(df[ts_col], errors="coerce")

    df["hour"] = df[ts_col].dt.hour
    df["day_of_week"] = df[ts_col].dt.dayofweek
    df["day"] = df[ts_col].dt.day
    df["month"] = df[ts_col].dt.month

    return df


def compute_rate(df: pd.DataFrame) -> pd.DataFrame:
    """
    Convert cumulative counter 'value' into a per-row rate (diff per instance/cpu/mode).
    """
    df = df.sort_values(by=[TIMESTAMP_COLUMN, "instance", "cpu", "mode"])
    df["value_rate"] = df.groupby(["instance", "cpu", "mode"])["value"].diff().fillna(0)
    return df


def split_features(
    df: pd.DataFrame,
    target: str,
) -> Tuple[pd.DataFrame, pd.Series]:
    """
    Separate features and target with explicit typing.
    """
    X: pd.DataFrame = df.drop(columns=[target])
    y = df[target]
    if isinstance(y, pd.DataFrame):
        y = y.squeeze()  # ensure Series
    return X, y


# ----------------------------
# Model Pipeline
# ----------------------------

def build_pipeline(
    numeric_features: List[str],
    categorical_features: List[str],
) -> Pipeline:
    """
    Build preprocessing + XGBoost pipeline.
    """
    preprocessor = ColumnTransformer(
        transformers=[
            ("num", "passthrough", numeric_features),
            ("cat", OneHotEncoder(handle_unknown="ignore"), categorical_features),
        ],
        remainder="drop",
    )

    model = XGBRegressor(
        n_estimators=300,
        max_depth=6,
        learning_rate=0.05,
        subsample=0.8,
        colsample_bytree=0.8,
        objective="reg:squarederror",
        n_jobs=-1,
        random_state=RANDOM_STATE,
    )

    pipeline = Pipeline(
        steps=[
            ("preprocessor", preprocessor),
            ("model", model),
        ]
    )

    return pipeline


# ----------------------------
# Training / Evaluation
# ----------------------------

def train_and_evaluate(df: pd.DataFrame) -> Pipeline:
    """
    Train XGBoost model and print evaluation metrics.
    """
    # Compute per-interval rate
    df = compute_rate(df)

    # Add time-based features
    df = add_time_features(df, TIMESTAMP_COLUMN)

    # Drop non-feature columns
    df = df.drop(
        columns=["created_at", TIMESTAMP_COLUMN, "value"],
        errors="ignore",
    )

    # Identify feature types dynamically
    numeric_features: List[str] = df.select_dtypes(include=["int64", "float64"]).columns.tolist()
    categorical_features: List[str] = df.select_dtypes(include=["object"]).columns.tolist()

    # Remove target from features
    if TARGET_COLUMN in numeric_features:
        numeric_features.remove(TARGET_COLUMN)

    X, y = split_features(df, TARGET_COLUMN)

    X_train, X_test, y_train, y_test = train_test_split(
        X,
        y,
        test_size=TEST_SIZE,
        random_state=RANDOM_STATE,
        shuffle=True,  # TODO: change to time-based split for forecasting
    )

    pipeline = build_pipeline(numeric_features, categorical_features)
    pipeline.fit(X_train, y_train)

    preds = pipeline.predict(X_test)

    mae: float = mean_absolute_error(y_test, preds)
    rmse: float = mean_squared_error(y_test, preds) ** 0.5

    print("Model evaluation:")
    print(f"MAE : {mae:.4f}")
    print(f"RMSE: {rmse:.4f}")

    return pipeline


# ----------------------------
# Entry Point
# ----------------------------

def main() -> None:
    df: pd.DataFrame = ingest_data.load_metrics_df(limit=5000)
    model: Pipeline = train_and_evaluate(df)

    joblib.dump(model, "xgb_capacity_model.pkl")
    print("Model saved to xgb_capacity_model.pkl")


if __name__ == "__main__":
    main()
