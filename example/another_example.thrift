namespace go another_example

struct AnotherExample {
    1: required i64 id,
    2: required string name,
    3: required string address,
    4: required i64 age,
}

struct GetAnotherExampleRequest {
    1: required i64 id (api.path="id"),
}

struct CreateAnotherExampleRequest {
    1: required string name (api.vd="regexp('[a-zA-Z]{3,}', $)"),
    2: required string address (api.vd="len($)<=255"),
    3: required i64 age (api.vd="$>=18"),
}

service AnotherExampleService {
    AnotherExample get(GetAnotherExampleRequest request) (api.get="/another-example/:id");
    AnotherExample create(CreateAnotherExampleRequest request) (api.post="/another-example");
}