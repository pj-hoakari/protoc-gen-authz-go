# protoc-gen-authz-go

`protoc-gen-connect-go` と同じ Go パッケージに、RPC ごとの認証認可レベルを検証する Connect interceptor と DI 用 handler コンストラクタを生成する内部用プラグイン

## Custom option

```proto
import "authz/v1/options.proto";

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

`authz/v1/options.pb.go` はこのモジュールに同梱されているため、利用側で custom option 用の Go コードを生成する必要はない  
利用側では `github.com/pj-hoakari/protoc-gen-authz-go/authz/v1` を Go module の依存に追加する  

`protoc` 用の schema は利用側のリポジトリで include root に書き出すことができる  

```sh
protoc-gen-authz-go --write-proto proto
# proto/authz/v1/options.proto が作成される
```

生成時には、書き出した `authz/v1/options.proto` のバージョンとプラグインが認識するバージョンの一致を検証する
不一致またはバージョン未宣言の schema は生成エラーになる

`protoc-gen-connect-go` と同時に実行

```sh
protoc -I. -I"$(protoc-gen-authz-go --proto-path)" \
  --go_out=. \
  --connect-go_out=. \
  --authz-go_out=. \
  api/greeting/v1/greeting.proto
```

## Develop

正規の schema は `proto/authz/v1/options.proto`。これを変更する場合は、`option (authz.v1.authz_proto_version)` とプラグインの認識バージョンも同じ新しい値へ更新し、`task proto:gen:go` で生成物を更新する

CI では 
- schema の全変更について、両方のバージョンが前のリビジョンより増えて一致することの検証
- `protoc-gen-authz-go --proto-version` でプラグインが認識するバージョンの取得・検証  

を行う
