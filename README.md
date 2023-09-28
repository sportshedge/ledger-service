A very light-weight double accounting ledger, written in Go for best performance.
Postgres is the database of choice here.
Reason:
     Ledger works heavily on transactions and uses relations as well to ensure no balance records are kept for books that are not there.
     Without transaction, managing balance updates are really difficult when something fails. Hence, db is not switchable, it's `Postgres`.

Features:
1. BookId 1 is company cashbook. (Not really mandated, but should be a mandate for better traceability)
2. Custom Environment var support by using `${VAR_NAME}`, VAR_NAME should be present in os.ENV.
3. Ledger supports custom json environment variables along with string env vars as well (key will be string only),
   for now `map[string]string`, `map[string]bool`, `map[string]map[string]string` and `map[string]map[string]bool` are supported in terms of json.
   Any value inside of `env`.yaml (ex: local.yaml) should have a value that looks like -> "${XXX}",
   it will be auto replaced when code runs, XXX should be present in the environment.
     1. `map[string]map[string]string`: `local.yaml` -> `server.ServiceTokenWhitelist`, in .env, if you write `SERVICE_TOKEN_WHITELIST={"user_module":{"read":"abc","write":"cde"}}`,
        it will be automatically parsed into map of maps, the inner map will be map of map of string.
     2. `map[string]string` is also supported. ENV: `SERVICE_TOKEN_WHITELIST={"user_module":"abc"}`, change `local.yaml` -> `server.ServiceTokenWhitelist` to be `map[string]string`.
     3. Similarly, for bool types. Key is always string, keep that in mind.
4. Double entry accounting, fast, stores book level balance by asset and operation. 
5. Asset agnostic, platform agnostic, stocks, crypto... possibilities are endless.
6. You can ignore specific bookIds for which you don't need the balance using `EXCLUDED_BALANCE_BOOK_IDS` env. Example would be ignoring cashbook. The value should be , seperated string. (ex: 1,-1,0)  `By Default, all book's balances will be stored`. 
7. Concurrent operations are already taken care of. No loading data onto memory to avoid balance mess up during heavy concurrent scenarios.
8. DB level check constraint on bookId to ensure no -ve `OVERALL` type balance for a book and a given asset. (bookId 1 is excluded here)
9. Operation level balance grouping available (op can be LIMIT_ORDER, MARKET_ORDER, DEPOSIT, WITHDRAW, TRADE etc.) where actual balance is denoted by `OVERALL` op type.
10. Can be extended for margin/leverage easily in case of a trading platform. 
11. BookId based grouping, each user should have two books, block and main book. Keep in mind, ledger server won't and shouldn't know if it's block or main book of a user.
12. No session or transaction level advisory locks to ensure the highest throughput.
13. Different trade types i.e. INTRA-DAY, QUARTERLY etc. can be supported using the metadata. 

Note: To get balance for a book, if operationType is not provided, OVERALL(operationType) balance is fetched.

Api Doc: Check the collections folder, you'll see the postman.json. Import this collection in postman. Requests have examples.

Configuration is manged via viper, create a config file with the `APP_ENV` value for that environment. .env is used by default only in `local/localhost` env, `DOT_ENV` if marked `enable` in any other environment,
that will also use .env.

In production if used with ecs and is dependent on dotenv, make sure to create dotenv and store it inside s3 or pass all the env variables to task definition.

To pass .env file entirely, This below part should be with the ecs task definition ->
```
"environmentFiles": [
  {
    "value": "arn:aws:s3:::s3_bucket_name/envfile_object_name.env",
    "type": "s3"
  }
]
```


### How to run the server locally? 
  1. create .env file ->
      ```
      APP_ENV = xx
      DOT_ENV=enable
      RUN_MODE = xx 
      DB_TYPE = xx
      DB_USER = xx
      DB_PASSWORD = xx
      DB_HOST = 127.0.0.1
      DB_PORT = 5432
      DB_NAME = xx
      DB_TABLE_PREFIX = xx
      DB_SSL_MODE = disable
      JWT_SECRET = xxxx
      EXCLUDED_BALANCE_BOOK_IDS = 1,2,3 # if not provided, will store every bookId in the balances table.
      SERVICE_TOKEN_WHITELIST={"user_module":{"read":"abc","write":"cde"}}
      ```
  2. Install dependencies -> `go mod tidy`
  3. Install below items (no example as these are os dependent, these need to be installed in `code build stage` as well for `deployments`) ->
     1. protoc
     2. protoc-gen-go
     3. protoc-gen-go-grpc
     4. make
  4. run `make ledger`
  5. run `./run.sh`

The server should now run and have auto reload.

### How to do deployments?
- During deployments, during the build stage, it's build tool's responsibility to generate the go proto code, as it will be required for the server to start. 
- Your build tool can run makefile or install proto to generate and copy that to dockerfile.
- Current Dockerfile neither has support for proto code generation nor it ever will.

Notes:
1. Book Create/update method will create a book if the name of the book doesn't exist else it will update the book.
2. It is the ledger client's responsibility to maintain uniqueness of the book. 
3. To ensure uniqueness of the books for a given account holder, ledger client should create debit/credit books based on uuid-v1. 

To manage different types of books (
     Exclude these book ids from balance roll up table to ensure minimal performance bottlenecks
     and ensure, we calculate company balances for time periods required. 
     RevenueBook might need entry inside balance roll up, that is a discussion for another time.
):

1. CashBook: `bookID:1` This is the main company book from where money would be transferred. (It can go to -ve, and that will denote the total spending)
2. RevenueBook: `bookID:2` (Any income earned, i.e, income from trade fees etc. should come here, and it can/should not go -ve.)
3. ThirdPartyVendorBook: `bookID:3` (Any payment to 3rd party vendors should come here)
4. ExpenseBook (LiabilityBook): `bookID:4` (any expense, i.e. buying laptop for employees, will look like a transaction from BookID:1 -> BookID:4)
5. AssetBook: `bookID:5` This is assetBook, whichever asset Company decides to buy. (trx: BookID:1 -> BookID:5)
6. TDSBook: `bookID:6` This is for storing the tds if we deduce any which we've to submit.
7. IncomeTaxBook: `bookID:7` This is for storing the income tax company has to pay.

So, End of the day, it will translate into ->
`Total Asset = Ⲉ(Liability Books) + Ⲉ(Equity Books)`

TODO:
1. Test cases. (Integration test added, modification required)
2. ~~BookId validation while creating operation.~~ (Done)
3. ~~Better Config, using yaml and viper.~~ (Done)
4. Example Ledger Client implementation to manage the ledger of a crypto trading org.
5. Customisable bookIds, based on type (asset or liability).
6. Reserve top 100 bookIds for company books. Migration to partition the balances table, such that below 100 ids should get in a specific partition, remaining should be partitioned based on hash.
7. Better file naming, code cleanup.
8. ~~Grpc support.~~ (Done)
