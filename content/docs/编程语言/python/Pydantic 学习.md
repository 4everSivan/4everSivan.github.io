---
title: "Pydantic 学习"
---

# Pydantic 学习 

[TOC]

## 1. 简介

Pydantic 是 Python 中最广泛使用的数据验证库。快速且可扩展，Pydantic 与你的 linters/IDE/大脑配合得很好。用纯的、标准的 Python 3.9+ 定义数据；用 Pydantic 进行验证。

### 1.1为什么要用 Pydantic?

- **由类型提示驱动** — 使用 Pydantic，模式验证和序列化由类型注解控制；学习内容更少，代码更少，并且与您的 IDE 和静态分析工具集成. [了解更多…](https://docs.pydantic.dev/latest/why/#type-hints)
- **速度** — Pydantic 的核心验证逻辑是用 Rust 编写的。因此，Pydantic 是 Python 中最快的数据验证库之一. [了解更多…](https://docs.pydantic.dev/latest/why/#performance)
- **JSON 模式** — Pydantic 模型可以生成 JSON Schema，便于与其他工具集成. [了解更多…](https://docs.pydantic.dev/latest/why/#json-schema)
- **严格**和**宽松**模式— Pydantic 可以在严格模式（数据不转换）或宽松模式（Pydantic 尝试将数据强制转换为正确的类型）下运行. [了解更多…](https://docs.pydantic.dev/latest/why/#strict-lax)
- **数据类**、**类型字典**等 — TypedDicts 等 — Pydantic 支持验证许多标准库类型，包括 `dataclass` 和 `TypedDict` . [了解更多…](https://docs.pydantic.dev/latest/why/#dataclasses-typeddict-more)
- **定制化** — Pydantic 允许自定义验证器和序列化器以多种强大的方式更改数据处理方式. [了解更多…](https://docs.pydantic.dev/latest/why/#customisation)
- **生态系统** — 在 PyPI 上，大约有 8,000 个包使用 Pydantic，包括像 *FastAPI*、*huggingface*、*Django Ninja*、*SQLModel* 和 *LangChain* 这样广受欢迎的库. [了解更多…](https://docs.pydantic.dev/latest/why/#ecosystem)
- **实战验证** — Pydantic 每月下载量超过 3.6 亿次，被所有 FAANG 公司和纳斯达克前 25 大公司中的 20 家使用。如果你正在尝试使用 Pydantic 做些什么，其他人可能已经做过类似的事情了. [了解更多…](https://docs.pydantic.dev/latest/why/#using-pydantic)

### 1.2 演示

创建一个继承自 `BaseModel` 的自定义类：

```python
from datetime import datetime
from pydantic import BaseModel, PositiveInt


class User(BaseModel):
    id: int  
    name: str = 'John Doe'  
    signup_ts: datetime | None  
    tastes: dict[str, PositiveInt]  

    
external_data = {
    'id': 123,
    'signup_ts': '2019-06-01 12:22',  
    'tastes': {
        'wine': 9,
        b'cheese': 7,  
        'cabbage': '1',  
    },
}

user = User(**external_data)  

print(user.id)  
#> 123
print(user.model_dump())  
"""
{
    'id': 123,
    'name': 'John Doe',
    'signup_ts': datetime.datetime(2019, 6, 1, 12, 22),
    'tastes': {'wine': 9, 'cheese': 7, 'cabbage': 1},
}
"""
```

验证失败：

```python
from datetime import datetime
from pydantic import BaseModel, PositiveInt, ValidationError

class User(BaseModel):
    id: int
    name: str = 'John Doe'
    signup_ts: datetime | None
    tastes: dict[str, PositiveInt]


external_data = {'id': 'not an int', 'tastes': {}}  

try:
    User(**external_data)  
except ValidationError as e:
    print(e.errors())
    """
    [
        {
            'type': 'int_parsing',
            'loc': ('id',),
            'msg': 'Input should be a valid integer, unable to parse string as an integer',
            'input': 'not an int',
            'url': 'https://errors.pydantic.dev/2/v/int_parsing',
        },
        {
            'type': 'missing',
            'loc': ('signup_ts',),
            'msg': 'Field required',
            'input': {'id': 'not an int', 'tastes': {}},
            'url': 'https://errors.pydantic.dev/2/v/missing',
        },
    ]
    """
```

## 2. 概念

### 2.1 模型

在 Pydantic 中定义模式的主要方式之一是通过模型。模型只是从 `BaseModel` 继承的类，并将字段定义为注解属性。可以将模型类比为 C 语言等语言中的结构体，或者 API 中单个端点的需求。**不受信任的数据**可以传递给模型，经过解析和验证后，Pydantic 保证结果模型实例的字段将符合模型上定义的字段类型。

### 2.2 字段（Field）与必填/可选

- 使用类型注解定义字段类型；带默认值为可选字段；`Optional[T]` 或 `T | None` 表示可为 `None`。
- 使用 `Field()` 指定默认值、别名、描述、范围等元数据。

```python
from typing import Optional
from pydantic import BaseModel, Field, constr, conint


class Product(BaseModel):
    id: int
    name: constr(min_length=1, max_length=50)
    stock: conint(ge=0) = 0
    description: Optional[str] = Field(None, description="可为空的商品描述")
    # 字段别名与序列化名
    external_id: str = Field(alias="extId")


p = Product(id=1, name="书籍", stock=10, extId="SKU-1")
assert p.external_id == "SKU-1"
# 反序列化支持别名；序列化时可选按别名输出
assert p.model_dump(by_alias=True)["extId"] == "SKU-1"
```

### 2.3 严格模式与类型转换

Pydantic 支持宽松解析（尝试把字符串等转换为目标类型）和严格模式：

```python
from pydantic import BaseModel, StrictInt


class M(BaseModel):
    # 使用 Strict* 类型或在配置中启用 strict
    count: StrictInt

M(count=1)           # OK
M(count="1")        # ValidationError: 严格整型不接受字符串
```

也可通过 `model_config = {"strict": True}` 对整个模型启用严格模式（v2）：

```python
from pydantic import BaseModel


class StrictModel(BaseModel):
    model_config = {"strict": True}
    x: int


# StrictModel(x=1) OK；StrictModel(x="1") 将失败
```

### 2.4 验证器（v2）

在 v2 中使用 `field_validator` 与 `model_validator`：

```python
from pydantic import BaseModel, field_validator, model_validator


class Signup(BaseModel):
    email: str
    password: str

    @field_validator("email")
    @classmethod
    def validate_email(cls, v: str) -> str:
        if "@" not in v:
            raise ValueError("非法邮箱")
        return v.lower()

    @model_validator(mode="after")
    def check_password_strength(self):
        if len(self.password) < 8:
            raise ValueError("密码太短")
        return self
```

`mode="before"` 可在解析前对原始输入进行预处理；`mode="after"` 在字段解析完成后对模型整体进行校验。

### 2.5 复杂类型与嵌套

支持嵌套模型、`list`/`set`/`dict`、`typing.Union`、`Literal`、`Annotated` 等：

```python
from typing import Annotated, Literal, Union
from pydantic import BaseModel, Field


class Address(BaseModel):
    city: str
    zip_code: Annotated[str, Field(min_length=5, max_length=10)]


class User(BaseModel):
    name: str
    role: Literal["admin", "user"] = "user"
    address: Address
    tags: list[str] = []
    contact: Union[str, int]  # 电话或邮箱字符串
```

### 2.6 序列化与 JSON Schema

- 使用 `model_dump()` 获取 `dict`；`model_dump_json()` 获取 JSON 字符串。
- `model_json_schema()` 生成 JSON Schema（便于文档/校验工具集成）。

```python
u = User(name="Alice", address={"city": "HZ", "zip_code": "310000"}, contact="alice@example.com")
data = u.model_dump()
json_str = u.model_dump_json()
schema = User.model_json_schema()
```

### 2.7 配置（v2：`model_config`）

常用配置键：

- `title`/`description`：用于 JSON Schema 元信息
- `populate_by_name`：允许用字段名/别名互相填充
- `from_attributes`：支持从对象属性读取（ORM 风格）
- `str_to_lower`、`str_strip_whitespace` 等字符串处理（通过 `field_validator` 更灵活）

```python
class ORMUser(BaseModel):
    model_config = {
        "from_attributes": True,
        "populate_by_name": True,
    }
    id: int
    username: str = Field(alias="user_name")
```

### 2.8 与数据类、设置管理互操作

- 数据类：`from pydantic import dataclasses as pydantic_dataclasses`
- 设置：`pydantic-settings` 提供 `BaseSettings`（单独包）

```python
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    debug: bool = False
    database_url: str

    class Config:
        env_prefix = "APP_"  # 读取 APP_DEBUG / APP_DATABASE_URL


settings = Settings()  # 自动从环境变量加载
```

## 3. 性能建议与最佳实践

- 使用精确的类型注解（减少歧义和回溯）。
- 大量实例化场景尽量避免多层嵌套与复杂 Union；必要时拆分模型或自定义轻量校验。
- 批量数据校验可结合生成器/分批处理，避免一次构建巨大对象。
- 在热点路径启用严格模式，提前失败，减少隐式转换开销。

## 4. 常见错误与排错

- “Field required”：字段缺失；添加默认值或使用 `Optional`。
- “Extra inputs are not permitted”：输入包含未声明字段；检查别名或允许 `extra`（v2 可用 `model_config={"extra":"allow"}`）。
- “Input should be a valid integer”：类型不匹配或严格模式触发；检查类型与转换。
- 模型嵌套时报错位置 `loc` 可帮助定位具体字段链路。

## 5. 与 v1 的差异与迁移要点（v1 → v2）

- `BaseModel.Config` → v2 使用 `model_config` 字典；部分键名变化。
- `validator`/`root_validator` → `field_validator`/`model_validator`；记得加 `@classmethod`（字段验证器）。
- `.dict()`/`.json()` → `.model_dump()`/`.model_dump_json()`。
- `parse_obj`, `parse_raw` 等老 API 在 v2 中被弃用或替换；首选构造函数与 `model_validate`。
- `allow_population_by_field_name` → `populate_by_name`。
- 更推荐使用 `typing.Annotated` 与 `Field` 结合做约束与元数据声明。

示例迁移：

```python
# v1
from pydantic import BaseModel, validator


class Item(BaseModel):
    name: str

    @validator("name")
    def non_empty(cls, v):
        assert v
        return v

    class Config:
        allow_population_by_field_name = True


# v2
from pydantic import BaseModel, field_validator


class Item(BaseModel):
    model_config = {"populate_by_name": True}
    name: str

    @field_validator("name")
    @classmethod
    def non_empty(cls, v: str) -> str:
        if not v:
            raise ValueError("name 不能为空")
        return v
```

更多迁移细节请参考官方指南（`https://docs.pydantic.dev/latest/migration/`）。

## 6. 快速入门

```python
# 解析任意输入并得到模型（失败抛 ValidationError）
User.model_validate({"name": "Ann", "address": {"city": "CD", "zip_code": "610000"}, "contact": 123})

# 仅校验，不构建实例（性能更优，需 pydantic-core 低层 API 或轻量自定义）
# 在应用中通常仍以模型为中心，便于维护与类型提示。

# 自定义序列化（模型字段转输出）
from pydantic import BaseModel, field_serializer


class Order(BaseModel):
    price_cents: int

    @field_serializer("price_cents")
    def to_yuan(self, v: int) -> float:
        return round(v / 100.0, 2)


o = Order(price_cents=1234)
assert o.model_dump()["price_cents"] == 12.34
```

