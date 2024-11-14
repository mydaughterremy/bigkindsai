import logging
import time
from typing import Any, Callable, Dict, Tuple


logger = logging.getLogger("data_ingestion")
logging.basicConfig(level=logging.INFO, format="[%(levelname)-5s] %(asctime)s %(funcName)s():%(lineno)d -- %(message)s")


def log_elapsed_time(func: Callable[..., Any]) -> Callable[..., Any]:
    """
    Decorator to measure elapsed time of function execution.
    ```
    @elapsed_time
    def func():
        pass

    func()
    # out: "func() Elapsed time: 0.82 s"
    ```
    """

    def timer(*args: Tuple[Any], **kwargs: Dict[str, Any]) -> Any:
        start: float = time.time()
        result: Any = func(*args, **kwargs)
        end: float = time.time()
        logger.info(f"{func.__name__}() Elapsed time: {(end - start):0.2f} s")
        return result

    return timer