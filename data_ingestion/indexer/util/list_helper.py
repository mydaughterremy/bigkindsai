from typing import List, Any


def _split_chunks(l: List[Any], n: int) -> List[List[Any]]:
    for i in range(0, len(l), n):
        yield l[i : i + n]
