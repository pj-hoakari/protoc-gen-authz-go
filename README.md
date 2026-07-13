# protoc-gen-authz-go

`protoc-gen-connect-go` と同じ Go パッケージに、RPC ごとの認証認可レベルを検証する Connect interceptor と DI 用 handler コンストラクタを生成する内部用プラグイン

## Custom option

```proto
import "authz/v1/authz.proto";

service GreeterService {
  option (authz.v1.service_auth_policy) = {
    level: AUTH_LEVEL_AUTHENTICATED
    required_scopes: "greeting.read"
  };

  rpc SayHello(SayHelloRequest) returns (SayHelloResponse) {
    option (authz.v1.auth_policy) = {
      level: AUTH_LEVEL_AUTHENTICATED
      required_scopes: "greeting.read"
    };
  }
}
```

RPC には `auth_policy`、service には `service_auth_policy` を設定できる  
解決順は RPC (Method) → Service → fail-closed  
未指定または `AUTH_LEVEL_UNSPECIFIED` は fail-closed で `AUTH_LEVEL_AUTHENTICATED` として生成される

## Generated API

各 service に次の API を生成

```go
type Verifier interface {
    Verify(context.Context, AuthPolicy) error
}

func NewGreeterServiceHandlerWithAuthz(
    svc GreeterServiceHandler,
    verifier Verifier,
    opts ...connect.HandlerOption,
) (string, http.Handler)
```

`Verifier` （JWT 検証・internal 判定・エラーコードの選択）はアプリケーション側で DI  
`PUBLIC` は verifier を呼ばない  
verifier が返した Connect error はそのまま返し、その他のエラーは `Unauthenticated` に正規化  

## Usage

利用側のリポジトリで include root を指定して schema を書き出す

```sh
protoc-gen-authz-go --write-proto proto
# proto/authz/v1/authz.proto が作成される
```

`protoc-gen-connect-go` と同時に実行

```sh
protoc -I. -I"$(protoc-gen-authz-go --proto-path)" \
  --go_out=. \
  --connect-go_out=. \
  --authz-go_out=. \
  api/greeting/v1/greeting.proto
```

実際の生成・DI・HTTP middleware との組み合わせは [`example/`](example/) を参照してください。
