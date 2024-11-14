from dataclasses import dataclass
from enum import Enum
from typing import Any, Dict, List, Optional

import dacite
import yaml


class IndexingType(Enum):
    WHOLE = "WHOLE"
    INCREMENTAL = "INCREMENTAL"


@dataclass
class TextField:
    type: str
    analyzer: Optional[str]
    src_field: List[str]
    fields: Optional[Dict[str, Any]]
    index: Optional[bool]


@dataclass
class SentenceTransformer:
    pass


@dataclass
class Encoder:
    sentenceTransformer: SentenceTransformer


@dataclass
class File:
    pass


@dataclass
class EmbeddingMethod:
    encoder: Optional[Encoder]
    file: Optional[File]


@dataclass
class VectorField:
    dimension: int
    src_field: List[str]
    ann_method: Dict[str, Any]
    embedding_method: EmbeddingMethod


@dataclass
class Field:
    text_field: Optional[Dict[str, TextField]]
    vector_field: Optional[Dict[str, VectorField]]


@dataclass
class Config:
    index_name: str
    settings: Dict[str, Any]
    fields: Field
    src_file: List[str]
    group_id: str
    job_id: str
    doc_id_fields: List[str]
    dedup_field: str
#     indexing_type: IndexingType = IndexingType.WHOLE
    batch_size: int = 2000


def parse_conf(path: str) -> Config:
    indexing_types = [indexingType.value for indexingType in IndexingType]
    with open(path) as f:
        d: dict = yaml.load(f, yaml.FullLoader)
        if type(d["batch_size"]) == str:
            d["batch_size"] = string_to_integer_transformer(d["batch_size"])
        if "indexing_type" in d:
            if d["indexing_type"] in indexing_types:
                d["indexing_type"] = IndexingType(d["indexing_type"])
            else:
                raise ValueError(f"indexing_type must be one of {indexing_types}")
        else:  # for backward compatibility
            d["indexing_type"] = IndexingType.WHOLE

        # empty for default
        d["src_file"] = []
        d["group_id"] = ""
        d["job_id"] = ""

        config: Config = dacite.from_dict(Config, d)
        return config


def string_to_integer_transformer(string: str) -> int:
    try:
        return int(float(string))
    except ValueError:
        return 2000  # default value
