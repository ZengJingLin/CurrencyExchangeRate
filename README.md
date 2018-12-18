# CurrencyExchangeRate

### config.json

    {"listen": "192.168.1.30:59420",  "statsCollectInterval": "5s"}
    
1. listen設定IP及Port

2. statsCollectInterval 為貨幣資料刷新間隔，有效的單位有"ns", "us" (or "µs"), "ms", "s", "m", "h"
   也可混合使用，例如:"2h45m"

### Running

    ./CurrencyExchangeRate config.json
    
### Visit

1. {currency}請輸入貨幣種類

2. {price}請輸入價格，價格僅接受數字

* http://192.168.1.30:59420/API/Insert/{currency}/{price}

* http://192.168.1.30:59420/API/Select/{currency}

* http://192.168.1.30:59420/API/Update/{currency}/{price}

* http://192.168.1.30:59420/API/Delete/{currency}