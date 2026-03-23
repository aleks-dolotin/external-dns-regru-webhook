# 1. Введение

Для повышения удобства технического взаимодействия клиентов и партнёров компании [«Регистратор доменных имён REG.RU»](https://www.reg.ru) с распределённой системой регистрации (далее RegRuSRS) был реализован простой и доступный программный интерфейс — REG.API, работающий поверх протокола HTTP. Мы думаем, что использование REG.API делает наше техническое взаимодействие с клиентами и партнёрами более эффективным.

Данное руководство описывает интерфейс доступа к REG.API второй версии, являющейся развитием API предыдущих версий, и предназначено для программистов, автоматизирующих взаимодействие с RegRuSRS.

Предполагается, что читатель знаком с основами HTTP и имеет навыки программирования.

## 1.1. Преимущества API 2.0 по сравнению с API 1.0

1. Унифицированная передача сложных структур данных.
2. Большая гибкость в выборе форматов передачи данных: возможность передачи входных параметров в форматах plain HTTP, JSON или XML, выходных — в виде JSON, YAML, XML или plain text.
3. Возможность параллельного выполнения нескольких нефинансовых операций одним пользователем.
4. Возможность работы с различными кодировками (по умолчанию utf8; также поддерживаются cp1251, koi8-r, koi8-u, cp866).
5. Многоязычные варианты ответов системы.
6. Выдача расширенной информации об ошибках.
7. Унифицированный способ идентификации доменов и услуг, с которыми производится операция.
8. Различные способы идентификации услуг: кроме имени домена, теперь вы можете использовать наши идентификаторы услуг в системе, которые позволяют всегда точно и легко указать нужную услугу.
9. Расширенные возможности отладки: различные тестовые функции, возможность просмотра входных параметров (с целью контроля правильности их передачи и декодирования).
10. Возможность выставлять в поле Content-type в ответах системы любое удобное для Вас значение.
11. Большая часть функций доступна и обычным пользователям! Вам необязательно быть партнёром!

[Презентация по преимуществам API 2.0.](https://www.reg.ru/docs/regru-api2-advantages.pdf)

## 1.2. Рекомендации по эффективному взаимодействию с API

За время более чем двухлетней эксплуатации REG.API был накоплен значительный опыт технического взаимодействия с партнёрами, выявлены типичные проблемы взаимодействия.

Одной из проблем, с которой сталкиваются партнёры, является превышение лимита запросов к REG.API (1200 запросов в час для клиента/ip). Оба лимита действуют единовременно. Если один из лимитов превышен, REG.API устанавливает код ошибки (в зависимости от типа лимита) в IP\_EXCEEDED\_ALLOWED\_CONNECTION\_RATE или ACCOUNT\_EXCEEDED\_ALLOWED\_CONNECTION\_RATE.

Анализ подобных ситуаций позволяет однозначно заключить, что проблема в подобных случаях заключается в неправильном / нецелевом использовании API в связи с ошибками либо архитектурными просчётами в программном обеспечении на стороне партнёров.

Считаем своим долгом дать ряд практических советов, которые позволят, с одной стороны, уменьшить вероятность превышения допустимого количества запросов к REG.API и следующего за этим временного блокирования операций партнёра и, с другой стороны, снизить нагрузку на систему регистрации RegRuSRS.

1. Рекомендуем осуществлять WHOIS-запросы по доменам (для отображения информации WHOIS на Ваших сайтах) не через REG.API, а обращаясь напрямую к WHOIS-серверам соответствующих доменных зон.

   При этом Вы получаете следующие преимущества:

   * ответ на WHOIS-запрос будет получен Вами быстрее,
   * предлагаемая схема более надёжна, поскольку исключаются лишние звенья,
   * уменьшается вероятность блокировки при превышении лимита запросов к API за счёт сокращения общего количества API-запросов.

   Мы предлагаем готовые программные решения, которые могут облегчить Ваши трудозатраты перехода на правильную схему реализации функционала для осуществления WHOIS-запросов.
2. Рекомендуем обращаться к API только для совершения заказов / изменения данных, но не для получения информации. Программное обеспечение некоторых наших партнёров либо не хранит, либо хранит неполную информацию о доменах в локальной базе данных. В результате эта информация очень часто динамически скачивается с нашей системы регистрации через функции domain\_list, service/get\_info, service/get\_details, domain/get\_nss и т. п. Рекомендуем хранить полную информацию о доменах и услугах локально и обращаться к REG.API только при необходимости изменения информации в реестре. В этом случае Ваша система будет работать быстрее и надёжнее, будет меньше зависеть от доступности нашей системы регистрации с Вашего сервера.
3. Рекомендуем выполнять все запросы на изменение данных асинхронно.

   Программное обеспечение некоторых наших партнёров осуществляет операции по регистрации доменов и услуг и изменению данных непосредственно в момент обработки HTTP-запроса от клиента. При этом если API-запрос не выполняется по каким-либо причинам сразу (отсутствие связи, превышение лимита запросов, блокировки параллельных запросов), то соответствующий запрос фактически теряется и клиент партнёра получает сообщение об ошибке.

   Подобная схема взаимодействия является крайне ненадёжной и в конечном итоге неудобной для Ваших клиентов.

   Рекомендуем осуществлять все запросы по заказу услуг / изменению данных асинхронно, через механизм очередей. В этом случае:

   * исключаются блокировки API-запросов (параллельно выполняется только один API-запрос, поскольку очередь можно обрабатывать в один поток);
   * в случае отсутствия связи запрос может повторяться, пока он не будет выполнен (таким образом существенно увеличивается надёжность системы);
   * в случае ошибок при обработке запросов (если REG.API вернул код ошибки) партнёр может решить проблему и повторить заявку, при этом клиент не получает лишних сообщений об ошибках — большинство проблем могут решаться партнёром самостоятельно, без ведома и участия клиента.
4. Используйте массовые операции по мере возможности. Многие методы поддерживают операции над несколькими услугами и доменами одновременно.
5. Рекомендуем выполнять на Вашей стороне логирование всех API-запросов и ответов. При этом в случае возникновения каких-либо проблем в процессе взаимодействия наличие журнала позволит гораздо эффективнее диагностировать проблемы на Вашей стороне, а также более грамотно обращаться в нашу службу технической поддержки, приводя выдержки из журналов, описывающие проблему.

## 1.3. Официальные клиентские библиотеки для работы с провайдерами REG.API

* Модуль Regru::API для языка Perl ([CPAN](http://search.cpan.org/dist/Regru-API/lib/Regru/API.pm), [GitHub](https://github.com/regru/regru-api-perl)).

# 2. Общее описание REG.API 2.0

## 2.1. Общий принцип взаимодействия

Транспортным протоколом для вызова функций REG.API является HTTP (HTTPS). В REG.API поддерживаются запросы методом POST, т.к. он не имеет ограничений на длину запроса и является более безопасным, с передачей параметров в form data.

ВНИМАНИЕ! HTTP GET и передача параметров в строке запроса DEPRECATED. Поддержка GET-запросов прекращена в связи с требованиями безопасности.

Каждый вызов является атомарным и синхронным, то есть все запросы независимы друг от друга. Также все операции являются синхронными: результат операции возвращается сразу же, нет промежуточных состояний при выполнении операции. Выбор в пользу такого способа взаимодействия был сделан для удобства подключения к REG.API со стороны клиентов.

Параллельная обработка вызовов доступна для запросов, не изменяющих балланс клиента на счёте провайдера REG.API.

## 2.2. Формат запроса

URL для вызова функций выглядит следующим образом:

```
https://api.reg.ru/api/regru2/<имя_категории_функции>/<имя_функции>
```

Таким образом, для каждой функции имеем собственный URL для её вызова (в API 1.0 URL для всех функций был одинаковым, а вызываемая функция идентифицировалась с помощью параметра `action`).

Практически все функции требуют дополнительных параметров для своего вызова.

### 2.2.1. Виды входных параметров

Передаваемые параметры можно разделить на несколько категорий:

* параметры аутентификации;
* параметры идентификации услуги;
* параметры управления работой API;
* параметры, специфичные для конкретной функции.

Из перечисленных четырех видов входных параметров обязательными практически для всех функций являются параметры аутентификации. Конкретный набор необходимых параметров варьируется от функции к функции и документирован в описании конкретных функций.

### 2.2.2. Передача входных параметров

Все дополнительные параметры, если они есть, можно передавать в виде стандартных HTTP-параметров POST, где передаваемые данные кодируются как x-www-form-urlencoded.

Пример передачи параметров через POST запрос:

```
https://api.reg.ru/api/regru2/nop
output_content_type=plain
password=test
username=test
```

При использовании curl необходимо значения всех параметров заключать в кавычки во избежание интерпретации shell-интерпретатором спецсимволов.

Пример запроса через curl:

`curl -X POST -H 'Content-Type: application/x-www-form-urlencoded' -d 'username=test&password=test&output_content_type=plain' 'https://api.reg.ru/api/regru2/nop'`

Параметры функции также можно передавать в форматах JSON или XML В этом случае все параметры, сериализованные в строку, передаются как один HTTP-параметр через поле input\_data.

Пример передачи сериализованных параметров с помощью curl через запрос POST:

`curl -X POST -H 'Content-Type: application/x-www-form-urlencoded' -d 'input_format=json&input_data=%7B%22username%22%3A%22test%22%2C%22password%22%3A%22test%22%2C%22output_content_type%22%3A%22plain%22%7D' 'https://api.reg.ru/api/regru2/nop'`

Пример вызова со сложной структурой можно посмотреть, например, в описании функции [set\_rereg\_bids](#domain_set_rereg_bids).

При необходимости передачи сложных структур данных использование JSON и XML — единственный способ их передать.

### 2.2.3. Форматы входных параметров

Передача данных возможна в нескольких форматах: как простые параметры HTTP-запроса POST, который далее условно будет называться «PLAIN», так и JSON и XML. Для наглядности их лучше рассмотреть на примерах передачи данных с ответом в JSON и выводом всех полученных данных, дополнительно указав для этого входной параметр `show_input_params = 1`. Чтобы показать возможность передачи списка данных, в запрос добавлен список `leftdata`, не несущий никакой функциональности.

#### 2.2.3.1. Формат «PLAIN» (простые параметры HTTP)

##### Поля input\_format и input\_data

отсутствуют

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/nop
domain_name=qqq.ru
leftdata=3
output_content_type=plain
password=test
show_input_params=1
username=test
```

##### Пример успешного ответа:

```
{
   "answer" : {
      "service_id" : "123456"
   },
   "input_params" : {
      "domain_name" : "qqq.ru",
      "leftdata" : [
         "1",
         "2",
         "3"
      ],
      "output_content_type" : "plain",
      "password" : "test",
      "show_input_params" : "1",
      "username" : "test"
   },
   "result" : "success"
}
```

#### 2.2.3.2. Формат JSON

##### Значение поля input\_format

json

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/nop
input_data={"output_content_type":"plain","show_input_params":1,"domain_name":"qqq.ru","leftdata":[1,2,3]}
input_format=json
password=test
username=test
```

##### Пример успешного ответа:

```
{
   "input_params" : {
      "show_input_params" : "1",
      "output_content_type" : "plain",
      "domain_name" : "qqq.ru",
      "password" : "test",
      "leftdata" : [
         "1",
         "2",
         "3"
      ],
      "username" : "test"
   },
   "answer" : {
      "service_id" : "123456"
   },
   "result" : "success"
}
```

#### 2.2.3.3. Формат XML

##### Значение поля input\_format

xml

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/nop
input_data=<opt%20domain_name="qqq.ru"%20output_content_type="plain"%20show_input_params="1"><leftdata>1</leftdata><leftdata>2</leftdata><leftdata>3</leftdata></opt>
input_format=xml
password=test
username=test
```

##### Пример успешного ответа:

```
{
   "input_params" : {
      "domain_name" : "qqq.ru",
      "show_input_params" : "1",
      "output_content_type" : "plain",
      "password" : "test",
      "input_format" : "xml",
      "leftdata" : [
         "1",
         "2",
         "3"
      ],
      "username" : "test",

   },
   "answer" : {
      "service_id" : "123456"
   },
   "result" : "success"
}
```

### 2.2.4. Общие входные параметры

Конкретный набор необходимых параметров варьируется от функции к функции, однако часть параметров применима ко всем или к большинству функций. Эти параметры описаны в данном разделе.

#### 2.2.4.1. Параметры для аутентификации

Эти параметры являются необходимыми для функций, требующих аутентификации. Это поля `username` + `password` либо `username` + `signature` (выбор варианта зависит от используемого [способа авторизации](#common_auth)). Если в настройках API загружен хотя бы один SSL-сертификат, то необходимо передавать его в заголовке каждого запроса к API при любом способе аутентификации. Примеры приведены ниже.

#### Аутентификация по паролю

| Параметр | Описание |
| --- | --- |
| username | Имя пользователя (login) в системе RegRuSRS. |
| password | Основной пароль пользователя в системе регистрации REG.RU, либо альтернативный пароль для API, который задаётся на страницах «[Настройки Партнёра](/reseller/details)» и «[Настройки API](/user/account/#/settings/api/)». |

Пример запроса:

[Показать пример](#)

Ещё примеры использования см. ниже.

#### Аутентификация по сигнатуре

| Параметр | Описание |
| --- | --- |
| username | Имя пользователя (login) в системе RegRu. |
| sig | Закодированная Base64 RSA сигнатура sha512 хеша, подписанная ssl-сертификатом |

Сигнатура (подпись) создаётся на основе передаваемых параметров. Текст для подписи — объединенный через символ "точка с запятой" отсортированный массив UTF-8 строк, полученных из значений передаваемых параметров. Ноль, пустая строка и неопределенное значение следует игнорировать. Не забудьте включить сюда также значения `username` Подпись осуществляется приватным RSA ключом, соответствующий которому сертификат был ранее загружен при настройках безопасности.

В каждом запросе необходимо в заголовке передавать публичный ключ сертификата.

В качестве примера рассмотрим получение данных следующим запросом:

```
https://api.reg.ru/api/regru2/domain/nop
username=test
output_content_type=plain
show_input_params=1
domain_name=qqq.ru
leftdata=1
leftdata=2
leftdata=3
```

В этом запросе участвуют значения параметров: test, test, plain, 1, qqq.ru, 1, 2, 3. Отсортируем их и объединим через точку с запятой, получив строку: "1;1;2;3;plain;qqq.ru;test;test", которую подписываем при помощи личного ключа в командной строке:

```
echo -n "1;1;2;3;plain;qqq.ru;test" | openssl dgst -sign my.key -sha512 | openssl enc -base64 -out my.sig
```

В результате мы получили в файле my.sig закодированную base64 подпись для вышеприведенного запроса. Обращаем внимание на параметр -n команды echo, чтобы в подписываемый текст не попал символ перевода строки.

Теперь отправим запрос на сервер при помощи curl:

```
curl -X POST -k --key my.key --cert my.crt -F 'sig=<my.sig' -F 'username=test' -F 'output_content_type=plain' -F 'show_input_params=\
1' -F 'domain_name=qqq.ru' -F 'leftdata=1' -F 'leftdata=2' -F 'leftdata=\
3' 'https://api.reg.ru/api/regru2/domain/nop'
```

Вместо логина test нужно использовать своё имя пользователя. Для логина test доступ к API по сигнатуре не активирован.

Готовую функцию генерации подписи [см. ниже](#common_auth_params_code_examples).

#### Аутентификация c дополнительным использованием SSL сертификата

Дополнительная проверка может использоваться при аутентификации по паролю, Для этого в настройках API должен быть загружен хотя бы один SSL-сертификат.  
При аутентификации по сигнатуре дополнительная проверка обязательная. Для дополнительной аутентификации используется тот же ssl-сертификат, что и для сигнатуры — пример генерации сертификата приведен выше. Для проверки SSL-сертификата, в каждом запросе необходимо в заголовке передавать публичный ключ сертификата.

Пример запроса с авторизацией по паролю при помощи curl:

```
curl -X POST -k --key my.key --cert my.crt -d 'username=test&password=test&output_content_type=\
plain&show_input_params=1&domain_name=qqq.ru&leftdata=1&leftdata=2&leftdata=\
3' 'https://api.reg.ru/api/regru2/domain/nop'
```

Пример запроса на сервер с авторизацией по сигнатуре при помощи curl [приведен выше](#common_auth_params_sig_cur).

Вместо test нужно использовать свой логин. Для логина test доступ к API с использованием SSL сертификата не возможен.

Общий пример php-кода, где приведены авторизация с использованием сигнатуры и SSL-сертификата. Вызов идет к методу проверки доступности домена для регистрации.

```
#!/usr/bin/php

<?php

$params =
[
    'dname'        => 'yayayayayaya.ru',
    'input_format' => 'plain',
    'username'     => 'test',
];

$sig_params = $params;
sort($sig_params);

$pkeyid = openssl_pkey_get_private("file:///path/my.key");
openssl_sign( implode(';', $sig_params), $sig, $pkeyid );

$params['sig'] = base64_encode($sig);
$url = 'https://api.reg.ru/api/regru2/domain/check';

$curl = curl_init();
curl_setopt($curl, CURLOPT_URL, $url);
curl_setopt($curl, CURLOPT_POST, true);
curl_setopt($curl, CURLOPT_SSLCERT, getcwd() . "/my.crt");
curl_setopt($curl, CURLOPT_SSLKEY, getcwd() . "/my.key");
curl_setopt($curl, CURLOPT_RETURNTRANSFER,true);
curl_setopt($curl, CURLOPT_SSL_VERIFYPEER, 1);
curl_setopt($curl, CURLOPT_POSTFIELDS, $params);

$result = curl_exec($curl);
curl_close($curl);
$result = json_decode(urldecode($result),true);
echo "\n";
print_r($result);
echo "\n";
?>
```

Пример Perl-кода c авторизацией по паролю.

```
#!/usr/bin/perl

use strict;
use warnings;

use LWP;

my $url = 'https://api.reg.ru';

print "login by password\n";

my $ua = LWP::UserAgent->new(
    parse_head => 0,
    timeout    => 5,
    keep_alive => 30,
);

my %pas_form = (
    username          => 'test',
    password          => 'test',
    show_input_params => '1',
);

my $response = $ua->post( $url . '/api/regru2/reseller_nop', \%pas_form );

if ( $response->code == 200 ) {
    print "req ok: code 200\n";
}
else {
    print "req fail: code " . $response->code . "\n";
    exit;
}

print "answer:\n" . $response->content . "\n";
```

Пример Perl-кода c авторизацией с использованием сигнатуры. Для отправки SSL-сертификата в заголовке запроса используется модуль Net::SSL (для старых версий модуле LWP).

```
#!/usr/bin/perl

use strict;
use warnings;

use MIME::Base64;
use JSON::XS;
use Crypt::OpenSSL::X509;
use Crypt::OpenSSL::RSA;
use LWP::UserAgent;
use Net::SSL;

my $url =       'https://api.reg.ru';
my $cert_file = '/path/to/my.crt';
my $cert_key =  '/path/to/my.key';

$ENV{PERL_LWP_SSL_VERIFY_HOSTNAME} = 0;
$ENV{HTTPS_CERT_FILE}              = $cert_file;
$ENV{HTTPS_KEY_FILE}               = $cert_key;

print "login by signature\n";

# for get RSA PRIVATE KEY from pem PRIVATE KEY need use openssl
open my $fh, '<', $cert_key or die "Can't open file $cert_key: $!";
my $private_key =  do { local $/; <$fh> };
close $fh;

my $ua = LWP::UserAgent->new(
    parse_head => 0,
    timeout    => 5,
    keep_alive => 30,
);

my %sig_form = (
    username          => 'test',
    show_input_params => '1',
    show_sig_params   => '1',
);

my $sigtext = make_text_for_sig( \%sig_form );

print "$sigtext\n";

my $rsa_priv = Crypt::OpenSSL::RSA->new_private_key( $private_key );

$rsa_priv->use_sha512_hash();
my $signature = $rsa_priv->sign( $sigtext );

my $sig = encode_base64( $signature );

print "$sig\n";

$sig_form{sig} = $sig;

my $response = $ua->post( $url . '/api/regru2/reseller_nop', \%sig_form );

if ( $response->code == 200 ) {
    print "req ok: code 200\n";
}
else {
    print "req fail: code " . $response->code . "\n";
}

print "answer:\n" . $response->content . "\n";

# functions for generation of a signature

sub make_text_for_sig {
    return join ';', sort( _make_text_for_sig(@_) );
}

sub _make_text_for_sig {
    my ( $d ) = @_;

    if ( ref $d eq 'ARRAY' ) {
        return map { _make_text_for_sig( $_ ) } @$d;
    }
    elsif ( ref $d eq 'HASH' ) {
        return map { _make_text_for_sig( $$d{$_} ) } grep { $_ ne 'sig' } keys %$d;
    }
    elsif ( not ref $d and $d ) {
        return $d;
    }

    return ();
}
```

Пример Perl-кода c авторизацией с использованием сигнатуры, вызов более сложного варианта — регистрация домена.

```
#!/usr/bin/perl

use strict;
use warnings;

use MIME::Base64;
use JSON::XS;
use Crypt::OpenSSL::X509;
use Crypt::OpenSSL::RSA;
use LWP::UserAgent;

my $url =       'https://api.reg.ru';
my $cert_file = '/path/to/my.crt';
my $cert_key =  '/path/to/my.key';

print "create using login by signature\n";

# for get RSA PRIVATE KEY from pem PRIVATE KEY need use openssl
open my $fh, '<', $cert_key or die "Can't open file $cert_key: $!";
my $private_key =  do { local $/; <$fh> };
close $fh;

my $ua = LWP::UserAgent->new(
    parse_head => 0,
    timeout    => 5,
    keep_alive => 30,
    ssl_opts => {
        SSL_cert_file => $cert_file,
        SSL_key_file  => $cert_key,
   },
);

my %cre_form = (
    username     => 'pppp',
    input_format => 'json',
    input_data   => {
        'contacts' => {
            'b_fax'          => '+7.9665443322',
            't_addr'         => 'Bolshoy sobachiy, 33',
            'a_state'        => 'Moskva',
            'o_fax'          => '+7.9665443322',
            'o_postcode'     => '291000',
            't_phone'        => '+7.9665443322',
            'a_fax'          => '+7.9665443322',
            'o_addr'         => 'Bolshoy sobachiy, 33',
            'a_email'        => 'sig@ya.ru',
            'a_addr'         => 'Bolshoy sobachiy, 33',
            't_company'      => 'FirmA, LLC',
            'b_phone'        => '+7.9665443322',
            'o_company'      => 'FirmA, LLC',
            'o_city'         => 'Moscow',
            'b_first_name'   => 'Ivan',
            't_city'         => 'Moscow',
            'b_state'        => 'Moskva',
            'a_postcode'     => '291000',
            't_first_name'   => 'Ivan',
            't_state'        => 'Moskva',
            't_email'        => 'sig@ya.ru',
            'a_country_code' => 'RU',
            'o_phone'        => '+7.9665443322',
            'a_city'         => 'Moscow',
            't_country_code' => 'RU',
            'b_last_name'    => 'Ivanoff',
            'a_phone'        => '+7.9665443322',
            't_postcode'     => '291000',
            't_last_name'    => 'Ivanoff',
            'b_addr'         => 'Bolshoy sobachiy, 33',
            'b_company'      => 'FirmA, LLC',
            'o_last_name'    => 'Ivanoff',
            'b_country_code' => 'RU',
            'a_company'      => 'FirmA, LLC',
            'b_email'        => 'sig@ya.ru',
            'a_last_name'    => 'Ivanoff',
            't_addr'         => 'Bolshoy sobachiy, 33',
            'b_city'         => 'Moscow',
            'o_state'        => 'Moskva',
            'o_first_name'   => 'Ivan',
            't_fax'          => '+7.9665443322',
            'b_postcode'     => '291000',
            'o_country_code' => 'RU',
            'o_email'        => 'sig@ya.ru',
            'a_first_name'   => 'Ivan'
        },
        'domain_name' => '', # not to register the domain and to receive a registration error
        'nss' => {},
        'enduser_ip' => '10.11.12.13',
    },
    show_input_params => '1',
    show_sig_params   => '1',
);

# or use this data:
# 'domains'    => [ { 'dname' => 'test.com' } ],
# 'nss'        => { 'ns0' => 'ns1.reg.ru', 'ns1' => 'ns2.reg.ru' },

my $sigtext = make_text_for_sig( \%cre_form );

print "$sigtext\n";

my $rsa_priv = Crypt::OpenSSL::RSA->new_private_key( $private_key );

$rsa_priv->use_sha512_hash();
my $signature = $rsa_priv->sign( $sigtext );

my $sig = encode_base64( $signature );

print "$sig\n";

$cre_form{sig} = $sig;

$cre_form{input_data} = JSON::XS->new->utf8->encode( $cre_form{input_data} );

my $response = $ua->post( $url . '/api/regru2/domain/create', \%cre_form );

if ( $response->code == 200 ) {
    print "req ok: code 200\n";
}
else {
    print "req fail: code " . $response->code . "\n";
    exit;
}

print "answer:\n" . $response->content . "\n";

# functions for generation of a signature

sub make_text_for_sig {
    return join ';', sort( _make_text_for_sig(@_) );
}

sub _make_text_for_sig {
    my ( $d ) = @_;

    if ( ref $d eq 'ARRAY' ) {
        return map { _make_text_for_sig( $_ ) } @$d;
    }
    elsif ( ref $d eq 'HASH' ) {
        return map { _make_text_for_sig( $$d{$_} ) } grep { $_ ne 'sig' } keys %$d;
    }
    elsif ( not ref $d and $d ) {
        return $d;
    }

    return ();
}
```

#### 2.2.4.2. Параметры для управления работой API

К дополнительным параметрам можно отнести общие параметры управления работой API и параметры идентификации услуги.

К общим параметрам управления работой API относятся функции управления форматом входных и выходных параметров функции, выбора рабочей кодировки и языка.

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| io\_encoding | Параметр позволяет явно (вместо стандартной utf8) указать кодировку, используемую для обмена данными (в данный момент поддерживаются cp1251, koi8-r, koi8-u, cp866). | utf8 |
| Управление входными данными | | |
| input\_format | Формат данных, передаваемый при вызове функции API. Сами данные при этом передаются с полем input\_data. На данный момент обрабатываются значения json и xml, любое другое приравнивается к значению по умолчанию, plain, и разбор данных не производится. | plain |
| input\_data | Данные в формате, указанном в input\_format. При этом данные аутентификации (username + password/signature) не должны передаваться внутри input\_data. | — |
| Управление выходными данными | | |
| output\_format | Параметр позволяет задать формат ответов системы — «json» (по умолчанию), «yaml», «xml» или «plain» (формат REG.API 1.0). В отношении YAML надо отметить, что данные отдаются закодированными в Base64. | json |
| view | Синоним для output\_format. Устарело. | json |
| output\_content\_type | Возможность задать content type, не изменяя формата ответа, для text/plain достаточно указать plain. | Зависит от значения поля output\_format. |
| lang | Язык текста ошибки error\_text, сейчас доступны русский, английский и тайский: ru, en, th. При этом код ошибки error\_code остаётся неизменным. По умолчанию текст ошибок: английский. | en |
| show\_input\_params | Возвращает все входные поля как хеш параметра input\_params, при этом если входные данные были в json или xml формате, то данные отображаются после обработки json- или xml-парсером. Т.е. если указать input\_format=json и output\_format=xml, то для input\_params будет сделано преобразование из JSON в XML. | 0 |

#### 2.2.4.3. Параметры для идентификации услуги

Параметры идентификации услуги требуются при выполнении операций над конкретной ранее заказанной услугой, когда её надо сначала идентифицировать.

Возможны следующие варианты идентификации:

1. по ID услуги (как для доменов, так и для услуг),
2. по имени домена (для доменов),
3. по имени домена и типу услуги (для услуг),
4. по ID родительской услуги, типу услуги и подтипу услуги (для услуг).

Наиболее точной и быстрой является идентификация по числовому идентификатору услуги, поэтому мы рекомендуем хранить на своей стороне и использовать ID домена/услуги и при вызовах передавать идентификатор услуги.

| Параметр | Описание |
| --- | --- |
| Идентификация по ID-услуги (рекомендуется) | |
| service\_id | Числовой идентификатор услуги. |
| Идентификация по ID-услуги, задаваемому пользователем | |
| user\_servid | Число-буквенный идентификатор услуги, для его использования надо сам идентификатор задать при создании услуги/домена. Получить ранее заданный идентификатор можно используя функцию [service/get\_info](#service_get_info). |
| Идентификация доменов по имени | |
| domain\_name | Имя домена. Русские имена доменов передаются в кодировке punycode либо в национальной кодировке. |
| Идентификация услуг по имени домена и типу услуги [(кроме VPS)](#except_vps) | |
| domain\_name | Имя домена, к которому привязана услуга. Русские имена доменов передаются в кодировке punycode либо в национальной кодировке. |
| servtype | Тип услуги. Например «srv\_hosting\_ispmgr» для хостинга или «srv\_webfwd» для услуги web-forwarding. |
| Идентификация услуг по ID родительской услуги, типу услуги и подтипу услуги | |
| uplink\_service\_id | ID родительской услуги, с которой связана искомая услуга. |
| servtype | Тип услуги. Например «srv\_hosting\_ispmgr» для хостинга или «srv\_webfwd» для услуги web-forwarding. |
| subtype | Подтип услуги. Например «pro» для лицензии ISP Manager Pro. |

#### 2.2.4.4. Параметры для идентификации списка услуг

| Параметр | Описание |
| --- | --- |
| domains | Список, каждый элемент которого содержит имя домена или его service\_id в соответствии со [стандартными параметрами идентификации услуги](#common_service_identification_params).  В одном запросе можно обратиться не более чем к 1000 услугам. |
| services | Список, каждый элемент которого содержит имя домена + тип сервиса или его service\_id в соответствии со [стандартными параметрами идентификации услуги](#common_service_identification_params). |

В общем виде input\_data в формате JSON для такого запроса выглядит так:

```
input_data={"services":[{"параметр_идентификации_услуги_1":"значение",...},
{"параметр_идентификации_услуги_2":"значение",...}]}
```

Ответ при запросе со списком услуг будет так же содержать список и для каждой услуги будет указан результат в поле `result`. В случае успеха это будет success, в случае ошибки — текст самой ошибки, а также поле error\_code со стандартным кодом ошибки, совпадающим с кодом при запросе 1 домена.

Пример: получение service\_id услуг используя service/nop с ошибками в двух последних значениях

[Показать пример](#)

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : 12345,
            "servtype" : "domain"
         },
         {
            "dname" : "test.su",
            "result" : "success",
            "service_id" : 12346,
            "servtype" : "srv_hosting_ispmgr"
         },
         {
            "dname" : "test12347.ru",
            "result" : "success",
            "service_id" : "111111",
            "servtype" : "domain"
         },
         {
            "error_code" : "INVALID_SERVICE_ID",
            "error_text" : "service_id is invalid",
            "result" : "error",
            "service_id" : "22bug22"
         },
         {
            "error_code" : "NO_DOMAIN",
            "error_text" : "domain_name not given or empty",
            "result" : "error",
            "surprise" : "surprise.ru"
         }
      ]
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

Ниже, при детальном описании каждой функции, будет указана поддержка обработки списка услуг.

#### 2.2.4.5. Параметры для идентификации папки

В данном разделе описаны способы идентификации папок.

Наиболее точной и быстрой идентификацией папок является идентификация по числовому идентификатору папки `folder_id`, поэтому мы рекомендуем хранить на своей стороне и использовать ID папки и при вызовах передавать числовой идентификатор папки.

| Параметр | Описание |
| --- | --- |
| folder\_id | Числовой идентификатор папки (рекомендуется). |
| folder\_name | Имя папки. |

#### 2.2.4.6. Параметры оплаты

В данном разделе описаны общие параметры для функций, связанных с заказом или продлением услуг, т. е. функций, которые задействуют оплату.

| Параметр | Описание |
| --- | --- |
| pay\_type | Способ оплаты счёта. На данный момент доступные такие варианты оплаты: (bank, pbank, prepay, yacard)  Значение по умолчанию: prepay.  Заметьте, что автоматически счёт может быть оплачен только при выборе способа оплаты prepay и наличии достаточного количества средств на лицевом счёте. В противном случае заявка будет помечена как неоплаченная и Вам нужно убдет вручную оплачивать её из «Личного кабинета». |
| ok\_if\_no\_money | Разрешает создавать счет, если денег для оплаты недостаточно. В этом случае заявка в системе создаётся, однако эта заявка будет исполнена только после выполнения операции «сменить способ оплаты счёта» через web-интерфейс системы. Если флаг не установлен и денег на счёте недостаточно - возвращается ошибка и заявка не создается. |

## 2.3. Формат ответа

### 2.3.1. Передача выходных параметров

Все функции могут возвращать ответы в форматах JSON, YAML, XML и plain text. По умолчанию используется JSON. Выходной формат передачи данных переключается с помощью опции [output\_format](#common_api_management_params).

Некоторые функции имеют дополнительные форматы вывода, помимо перечисленных. Например, функция [get\_rereg\_data](#domain_get_rereg_data) может отдавать данные в CSV формате.

#### 2.3.1.1. Формат JSON

##### Значение поля output\_format

* json

##### Примеры запросов:

```
https://api.reg.ru/api/regru2/nop
password=test
username=test
```

```
https://api.reg.ru/api/regru2/nop
output_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "login" : "test",
      "user_id" : 0
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

#### 2.3.1.2. Формат YAML

##### Значение поля output\_format

* yaml

##### Пример запроса:

```
https://api.reg.ru/api/regru2/nop
output_format=yaml
password=test
username=test
```

##### Пример ответа:

```
---
answer:
  login: test
  user_id: 0
charset: utf-8
messagestore: !!perl/hash:SRS::MessageStore
  _messages: {}

result: success
```

#### 2.3.1.3. Формат XML

##### Значение поля output\_format

* xml

##### Пример запроса:

```
https://api.reg.ru/api/regru2/nop
output_format=xml
password=test
username=test
```

##### Пример ответа:

```
<opt charset="utf-8" result="success">
  <answer login="test" user_id="0" />
  <messagestore name="_messages" />
</opt>
```

#### 2.3.1.4. Формат PLAIN

В данном формате ответ возвращается в упрощённом виде: не возвращаются сложные / вложенные структуры данных. В связи с данным ограничением не рекомендуется использовать этот формат.

##### Значение поля output\_format

* plain

##### Пример запроса:

```
https://api.reg.ru/api/regru2/nop
output_format=plain
password=test
username=test
```

##### Пример ответа:

```
Success: ; login: test; user_id: 0
```

### 2.3.2. Общие выходные параметры

Все ответы API функций (как ошибки, так и успешные ответы) стандартизованы.

Обязательным полем для любого ответа является result. Оно может иметь значения "error" или "success".

Поля, присутствующие в положительном ответе:

| Поле | Описание |
| --- | --- |
| result | Имеет значение success. |
| answer | Хеш, содержащий результат работы функции. |
| input\_params | Хеш с параметрами, переданными при вызове функции. Присутствует, если вызов был с `show_input_params=1`. |

Пример положительного ответа:

```
{
   "answer" : {
      "login" : "test",
      "user_id" : "0"
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

Поля, возвращаемые в случае ошибки:

| Поле | Описание |
| --- | --- |
| result | Имеет значение error. |
| error\_code | Код ошибки, который представляет собой предложение в верхнем регистре с использованием "\_" в качестве разделителей и является уникальным внутри системы. Предназначен для анализа ошибок на уровне программ. Для пользователей создано поле error\_text. |
| error\_text | Подробное описание ошибки на английском или русском, в зависимости от входного параметра lang. |
| error\_params | Параметры, подставляемые в стандартный текст ошибки, могут быть полезны при автоматическом разборе ошибок. |
| input\_params | Хеш с параметрами, переданными при вызове функции. Присутствует, если вызов был с show\_input\_params=1. |

Пример ответа, возвращающего ошибку:

```
{
   "error_code" : "PASSWORD_AUTH_FAILED",
   "error_text" : "Username/password incorrect",
   "result" : "error"
}
```

Обращаем Ваше внимание на то, что тексты в поле error\_text не предназначены для автоматической обработки и могут быть изменены без дополнительного уведомления с нашей стороны. Не рекомендуем использовать данное поле для автоматической обработки результатов выполнения запросов.

Также следует обратить внимание что существует 2 уровня ошибок: 1 - уровень всей команды, 2 - уровень конкретных услуг в командах, которые оперируют не одной, а несколькими услугами.

Например, в нижеприведенном ответе первый уровень был выполнен успешно, а во втором уровне успех зависел от идентификатора услуги:

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : 12345,
            "servtype" : "domain"
         },
         {
            "dname" : "test.su",
            "result" : "success",
            "service_id" : 12346,
            "servtype" : "srv_hosting_ispmgr"
         },
         {
            "dname" : "test12347.ru",
            "result" : "success",
            "service_id" : "111111",
            "servtype" : "domain"
         },
         {
            "error_code" : "INVALID_SERVICE_ID",
            "error_text" : "service_id is invalid",
            "result" : "error",
            "service_id" : "22bug22"
         },
         {
            "error_code" : "NO_DOMAIN",
            "error_text" : "domain_name not given or empty",
            "result" : "error",
            "surprise" : "surprise.ru"
         }
      ]
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

В целом команда выполнена успешно, о чем говорит результат "success", но для услуг (доменов) "22bug22" и "surprise.ru" были возвращены коды ошибок "INVALID\_SERVICE\_ID" и "NO\_DOMAIN", соответственно.

То есть в отличии от ответов на простые команды:

```
{
  "service_id" : "22bug22",
  "result" : "service_id is invalid",
  "error_code" : "INVALID_SERVICE_ID"
}
```

и

```
{
  "surprise" : "surprise.ru",
  "result" : "domain_name not given or empty",
  "error_code" : "NO_DOMAIN"
}
```

для команд, работающих с несколькими услугами, необходимо также проверять результаты и коды ошибок по каждой услуги отдельно:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "ya.ru",
            "error_code" : "DOMAIN_ALREADY_EXISTS",
            "result" : "Domain already exists, use whois service"
         },
         {
            "dname" : "yayayayayaya.ru",
            "result" : "Available"
         },
         {
            "dname" : "xn--000.com",
            "error_code" : "INVALID_DOMAIN_NAME_PUNYCODE",
            "result" : "Invalid punycode value for domain_name"
         }
      ]
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

### 2.3.3. Общие коды ошибок

## Ошибки запроса

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| ONLY\_POST\_ALLOWED | Only HTTP method POST allowed | Допускается только HTTP метод POST. |
| QUERY\_PARAMS\_DISALLOWED | Query string parameters are disallowed. Pass parameters in form data. | Передача параметров в строке запроса не допускается. Необходимо передавать параметры в form data. |

## Ошибки авторизации

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| NO\_USERNAME | No username given | Не указано имя пользователя |
| NO\_AUTH | No authorization mechanism selected | Не определён способ авторизации |
| PASSWORD\_AUTH\_FAILED | Username/password Incorrect | Ошибка аутентификации по паролю |
| RESELLER\_AUTH\_FAILED | Only resellers can access to this function | Только партнёры имеют доступ к этой функции |
| ACCESS\_DENIED | Your access to API denied. Please, contact us | Ваш доступ к API заблокирован, обратитесь, пожалуйста, в тех.поддержку |
| PURCHASES\_DISABLED | Purchases disabled for this account | Покупки/заказы для этого аккаунта запрещены |
| ACCESS\_DENIED\_FROM\_IP | Access to API from this IP denied | Доступ к API с этого IP заблокирован |
| ACCOUNT\_BLOCKED | Account is blocked. The instructions on how to retrieve access can be found in Profile Page. | Аккаунт заблокирован. Инструкции по восстановлению доступа можно получить в Личном Кабинете. |
| USER\_AUTHENTICATION\_FAILED | Login or password is incorrect. | Неверная почта/логин или пароль. |
| MORE\_THAN\_ONE\_ACCOUNT\_WITH\_THE\_SAME\_EMAIL | More than one account with the same e-mail found. Please provide a login. | Обнаружено несколько аккаунтов с указанным e-mail адресом. Пожалуйста, введите логин. |

## Ошибки идентификации доменов, сервисов, папок

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| DOMAIN\_NOT\_FOUND | Domain $domain\_name not found or not owned by You | Домен $domain\_name не найден или вы не являетесь его владельцем |
| SERVICE\_NOT\_FOUND | Service $servtype for ext domain $domain\_name not found | Услуга $servtype для домена $domain\_name не найдена |
| SERVICE\_NOT\_SPECIFIED | Service identification failed | Ошибка идентификации сервиса |
| SERVICE\_ID\_NOT\_FOUND | Service $service\_id not found or not owned by You | Услуга с $service\_id не найдена или вы не является её владельцем |
| NO\_DOMAIN | domain\_name not given or empty | domain\_name не указано или пустое |
| INVALID\_DOMAIN\_NAME\_FORMAT | $domain\_name is invalid or unsupported zone | Формат $domain\_name неверен или необслуживаемая зона |
| INVALID\_SERVICE\_ID | service\_id is invalid | Формат service\_id неверен |
| INVALID\_DOMAIN\_NAME\_PUNYCODE | Invalid punycode value for domain\_name | Значение punycode для $domain\_name не верно |
| BAD\_USER\_SERVID | Invalid value for user\_servid. | Недопустимое значение для user\_servid. |
| USER\_SERVID\_IS\_NOT\_UNIQUE | Field servid is not unique | Поле servid с таким значением уже есть в системе |
| TOO\_MANY\_OBJECTS\_IN\_ONE\_REQUEST | Too many objects (more than $max\_objects) specified in one request | Указано слишком много объектов (более $max\_objects в одном запросе) |
| CLIENT\_NOT\_FOUND | Customer $client\_uid is not found | Клиент с идентификатором $client\_uid не найден |
| DOMAIN\_PROTECTION | Domain protection turn on for this domain or service. API operations disabled | Для данного домена или сервиса включена защита домена. Операции через API запрещены |
| NOT\_ALLOWED\_ORDER\_BY\_API | Service can't be ordered by API. It's only able in Account | Сервис не может быть заказан через API. Заказ возможен только через аккаунт |
| PARTCONTROL\_GRANT\_DISABLED | Partial control is disabled for this service | Передача данного сервиса в частичное управление запрещена |

## Ошибки доступности

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| SERVICE\_UNAVAILABLE | Reg.API is temporarily unavailable | Рег.API временно недоступен |
| BILLING\_LOCK | You have another active connection for billing operation to Reg.API | У вас активно другое соединение c Рег.API, связанное с биллинговыми операциями |
| DOMAIN\_BAD\_NAME | Domain $domain\_name is reserved or disallowed by the registry, or is a premium domain offered by special price | Домен $domain\_name является зарезервированным или недопустимым к регистрации по правилам реестра, либо premium-доменом, предлагаемым по специальной цене |
| DOMAIN\_BAD\_NAME\_ONLYDIGITS | Domain names that contains only digits can not be registered in this zone | Регистрация доменов, имя которых состоит только из цифр, в данной зоне не допускается |
| HAVE\_MIXED\_CODETABLES | You can't mix latin and cyrillic letters in domain names | Недопустимо смешивать кириллические и латинские буквы в имени домена |
| DOMAIN\_BAD\_TLD | Registration in $tld TLD is not available | Регистрация доменов в зоне .$tld не доступна |
| TLD\_DISABLED | Registration in $tld TLD is not available | Регистрация доменов в зоне .$tld не доступна |
| DOMAIN\_NAME\_MUSTBEENG | Russian letters are not allowed in chosen TLD (.$tld) | Русские буквы недопустимы в названии домена для выбранной зоны (.$tld) |
| DOMAIN\_NAME\_MUSTBERUS | Latin letters are not allowed in chosen TLD (.$tld) | Латинские буквы недопустимы в названии домена для выбранной зоны (.$tld) |
| DOMAIN\_ALREADY\_EXISTS | Domain already exists, use whois service | Домен уже существует, проверьте через whois |
| DOMAIN\_INVALID\_LENGTH | Invalid domain name length, You have entered too short or too long name | Недопустимая длина имени домена, вы ввели либо слишком короткое, либо слишком длинное имя |
| DOMAIN\_STOP\_LIST | Domain is unavailable, this domain name is either reserved or this is premium-domain with special price | Недоступное имя, Этот домен является зарезервированным, либо premium-доменом, предлагаемым по специальной цене |
| DOMAIN\_STOP\_PATTERN | Unfortunately domain name ($domain\_name) can't be registered | К сожалению, имя ($domain\_name) невозможно зарегистрировать |
| FREE\_DATE\_IN\_FUTURE | Domain freeing date is in the long time future | Дата освобождения домена $domain\_name наступает в будущем, ПОСЛЕ следующей даты массового освобождения доменов |
| NO\_DOMAINS\_CHECKED | You have chosen no domains for registration | Вы не выбрали ни одного домена для регистрации |
| NO\_CONTRACT | Filing preschedule for the domain registration after freeing is impossible before You signing up the contract on the domain registration | Подача ДОСРОЧНОЙ заявки на регистрацию домена после освобождения невозможна до заключения вами договора о регистрации доменов |
| INVALID\_PUNYCODE\_INPUT | Invalud Punycode name (error while converting from punycode) | Неверно заданное имя в punycode (ошибка при попытке перекодировки из Punycode) |
| CONNECTION\_FAILED | Domain check failed: can't connect to server. Please, try again later | Не удалось проверить состояние домена: невозможно установить соединение. Попробуйте повторить попытку позднее |
| DOMAIN\_ALREADY\_ORDERED | The domain name $domain\_name was order by you, You can pay for the registration and domain will be registered | Доменное имя $domain\_name уже заказано вами ранее к регистрации, вы можете оплатить его и заявка на регистрацию будет исполнена |
| DOMAIN\_EXPIRED | Domain $domain\_name is either expired or will expire in near future | К сожалению, срок делегирования домена $domain\_name либо уже истёк, либо истекает в ближайшее время |
| DOMAIN\_TOO\_YOUNG | From registration date of domain $domain\_name passed less than 60 days. Please, try to transfer domain later | К сожалению, с момента регистрации домена $domain\_name прошло менее 60-ти дней, попробуйте перенести домен позже |
| CANT\_OBTAIN\_EXPDATE | Can't determine expiration date of domain $domain\_name | Невозможно определить дату окончания делегирования домена $domain\_name |
| DOMAIN\_CLIENT\_TRANSFER\_PROHIBITED | Domain $domain\_name prohibited for transfer, contact previous registrar to unlock domain transfer | Домен $domain\_name запрещён к переносу, cвяжитесь с предыдущим регистратором для разблокирования домена |
| DOMAIN\_TRANSFER\_PROHIBITED\_UNKNOWN | Domain $domain\_name transfer prohibited, contact our technical support staff for details | Домен $domain\_name запрещён к переносу вышестоящим регистратором, cвяжитесь со службой технической поддержки для выяснения подробностей |
| DOMAIN\_REGISTERED\_VIA\_DIRECTI | Automatical internal transfers are unavailable in present time | Автоматический перенос доменного имени $domain\_name внутри DirectI запрещен |
| NOT\_FOUND\_UNIQUE\_REQUIRED\_DATA | Not found all data for check unique: dname, servtype or user\_id | Не найдены данные для проверки уникальности: dname, servtype или user\_id |
| ORDER\_ALREADY\_PAYED | Order on $dname $servtype is already payed | Заказ $dname $servtype уже оплачен ранее $ssru |
| DOUBLE\_ORDER | You already have not payed order on $dname $servtype | У вас уже есть неоплаченный заказ на $dname $servtype |
| DOMAIN\_ORDER\_LOCKED | The order or renew of the domain is disabled since processing of other operation for the same domain isn't completed yet | Заказ или продление домена заблокировано т.к. ещё не завершена другая операция на этот же домен |
| INSTALMENT\_NOT\_ALLOWED\_FOR\_SU | Instalment is not allowed for .SU | Частичная оплата недоступна для зоны .SU |
| UNAVAILABLE\_DOMAIN\_ZONE | $tld is unavailable domain zone | $tld не относится к списку поддерживаемых доменных зон |

## Ошибки при работе с DNS-зонами

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| DOMAIN\_IS\_NOT\_USE\_REGRU\_NSS | This domain not use REG.RU name services: ns1.reg.ru and ns2.reg.ru | Этот домен не использует DNS-сервера: ns1.reg.ru и ns2.reg.ru |
| REVERSE\_ZONE\_API\_NOT\_SUPPORTED | Reverse zone not supported now | Настройка реверсных зон на данный момент не поддерживается |
| IP\_INVALID | Invalid IP address | Ошибка в IP адресе |
| SUBD\_INVALID | Invalid subdomain | Неверный поддомен |
| CONFLICT\_CNAME | Can not set CNAME record together with other record for one subdomain | Для одного поддомена нельзя указывать записи CNAME совместно с другими записями |
| UNSUPPORTED\_UNICODE\_SYMBOL |  |  |

## Ошибки при работе с DNSSEC

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| DOMAIN\_DOESNT\_SUPPORT\_DNSSEC | This domain doesn't support DNSSEC | Этот домен не поддерживает DNSSEC |
| CANT\_GET\_INFO\_FROM\_REGISTRY | Can't get information from provider's registry | Невозможно получить информацию из реестра провайдера |
| DNSSEC\_UPDATING\_IN\_PROGRESS | DNSSEC updating for domain already in progress | DNSSEC для домена уже находится в процессе обновления |
| DOMAIN\_USES\_REGRU\_NSS | This domain uses REG.RU nameservers | Этот домен использует DNS-сервера Рег.ру |
| DOMAIN\_IS\_NOT\_USE\_REGRU\_NSS | This domain not use REG.RU name services: ns1.reg.ru and ns2.reg.ru | Этот домен не использует DNS-сервера: ns1.reg.ru и ns2.reg.ru |
| INVALID\_RECORDS | Invalid records | Неверные записи |

## Другие ошибки

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| NO\_SUCH\_COMMAND | Command $command\_name not found | Команда $command\_name не найдена |
| HTTPS\_ONLY | Access to api over non secure interface (http) prohibited! Please use https only | Доступ к API по небезопасному интерфейсу (http) запрещён! Используйте, пожалуйста, https |
| PARAMETER\_MISSING | $param required | $param не найден(ы) |
| PARAMETER\_INCORRECT | $param has incorrect format or data | $param имеет неверный формат или данные |
| NOT\_ENOUGH\_MONEY | Not enough money at account for this operation | Недостаточно денег для этой операции |
| INTERNAL\_ERROR | Internal error: $error\_detail | Внутренняя ошибка: $error\_detail |
| SERVICE\_OPERATIONS\_DISABLED | Operations on this service disabled | Операции с услугой запрещены |
| UNSUPPORTED\_CURRENCY | Unsupported currency | Валюта не поддерживается в системе |
| PRICES\_NOT\_FOUND | Prices not found for servtype $servtype | Для $servtype цены не найдены |
| SERVICE\_ALREADY\_EXISTS | Service already exists | Сервис уже существует |
| DOMAIN\_IS\_NOT\_ACTIVE | This domain is not activated yet | Домен не активирован |
| DOMAIN\_IS\_NOT\_VERIFIED | Email of the domain name has not been verified | Email домена не верифицирован |

## 2.4. Аутентификация

Если не добавлено ни одного ip-адреса, то работа с API невозможна. Добавить ip-адрес в список разрешенных можно в [«Настройках API»](/user/account/#/settings/api/).

Для доступа к большей части API функций требуется проведение аутентификации. Возможны следующие способы:

* По логину и паролю.
* По логину и сигнатуре
* Дополнительная проверка ssl-сертификата для способов, приведенных выше

### 2.4.1. Аутентификация по паролю

Для доступа по логину/паролю оба поля передаются в явном виде как значения для `username`, `password`:

```
https://api.reg.ru/api/regru2/nop
password=test
username=test
```

Список полей с описанием, используемых для аутентификации, перечислен в разделе [«Параметры для аутентификации / Аутентификация по паролю»](#common_auth_params_password_auth).

Возможные ошибки аутентификации см. в [стандартных кодах ошибок](#common_errors).

### 2.4.2. Аутентификация по сигнатуре

Для доступа по логину/сигнатуре оба поля передаются в явном виде как значения для `username`, `sig`.

Список полей с описанием, используемых для аутентификации, перечислен в разделе [«Параметры для аутентификации / Аутентификация по сигнатуре»](#common_auth_params_sig_auth).

Возможные ошибки аутентификации см. в [стандартных кодах ошибок](#common_errors).

### 2.4.3. Дополнительная проверка по ssl-сертификату

Дополнительная проверка может использоваться при аутентификации по паролю, Для этого в настройках API должен быть загружен хотя бы один SSL-сертификат. При аутентификации по сигнатуре дополнительная проверка обязательная.

Список дополнительных параметров запроса с описанием, перечислен в разделе [«Параметры для аутентификации / Аутентификация c дополнительным использованием SSL сертификата»](#common_auth_params_ssl_auth).

## 2.5. Тестовый и рабочий (боевой) доступ к REG.API

Для отладки работы с API предусмотрен тестовый доступ; для этого имя пользователя и пароль должны иметь значение «test».

При таком режиме работы осуществляются все проверки входных параметров, выдаётся ответ, но никаких действий не производится, и деньги за операции не снимаются. Ответ сервера при работе в тестовом режиме носит ознакомительный характер, в некоторых случаях может содержать неактуальные значения и не соответствовать реальной картине, имеющей место при использовании боевого доступа. При вызове функций в тестовом режиме никаких реальных данных о доменах не возвращается.

Также для отладочных целей есть несколько специализированных функций, которые предназначены для вызовов под реальными идентификационными данными. Это `nop`, `reseller_nop`, `user/nop`, `bill/nop`, `domain/nop`, `zone/nop`, `service/nop`, `folder/nop`. Они не выполняют никаких действий, но позволяют проверить доступность системы без её дополнительной нагрузки и, соответственно, с минимальным временем отклика. Ниже каждая из этих функций описана подробнее.

# 3. Общее описание функций

## 3.1. Виды функций

Все функции API в данный момент делятся на пять видов:

1. Пользовательские функции (им соответствует относительный путь user/) предназначены для получения данных, тем или иным образом связанных с конкретным пользователем (запросы баланса, статистики регистраций и проч.).
2. Функции работы с доменами (им соответствует относительный путь domain/) позволяют производить различные манипуляции с доменами.
3. Функции работы с услугами (им соответствует относительный путь service/) содержат всё необходимое для осуществления операция с услугами.
4. Функции работы с папками (им соответствует относительный путь folder/) позволяют группировать домены и услуги по собственным критериям.
5. Несколько отладочных функций вне категорий.

В зависимости от того, какая функция Вам нужна, требуется указывать различные адреса. Общий вид адреса такой:

https://api.reg.ru/api/regru2/<имя\_категории>/

где имя\_категории может принимать значения: user, domain, service, folder, bill, zone.

## 3.2. Доступность функций

Все функции REG.API условно можно разделить на три категории по доступности.

Первая категория — это общедоступные функции, при вызове которых не требуется указывать username. Как правило, это функции получения общих сведений, которые не зависят от того, кто их вызвал. В нижеприведенной таблице у таких функций графа «доступность» имеет значение «все».

Другая категория — это функции требующие аутентификации, но доступные для всех клиентов, зарегистрировавшихся на нашем сайте.

Третья категория ограничена клиентами, заключившими с нами партнёрское соглашение.

## 3.3. Список функций

Здесь приведен перечень доступных функций с кратким описанием и указанием их доступности. Полное, детальное описание каждой функции с примерами использования см. ниже.

| Функция | Краткое описание | Доступность |
| --- | --- | --- |
| Функции общего назначения | | |
| [nop](#common_nop) | возвращает login | Клиенты |
| [reseller\_nop](#common_reseller_nop) | возвращает login | Партнёры |
| [get\_user\_id](#common_get_user_id) | получение идентификатора авторизованного пользователя | Клиенты |
| [get\_service\_id](#common_get_service_id) | возвращает service\_id домена или услуги | Клиенты |
| Функции для работы с учётной записью | | |
| [user/nop](#user_nop) | тестовая функция | Все |
| [user/create](#user_create) | регистрация нового пользователя | Партнёры |
| [user/get\_statistics](#user_get_statistics) | получение статистики по пользователю | Клиенты |
| [user/get\_balance](#user_get_balance) | просмотр баланса | Клиенты |
| [user/set\_reseller\_url](#user_set_reseller_url) | установка URL для редиректа из внешней услуги | Партнёры |
| [user/get\_reseller\_url](#user_get_reseller_url) | получение URL для редиректа из внешней услуги | Партнёры |
| Функции для работы cо счетами (категория bill) | | |
| [bill/nop](#bill_nop) | тестовая функция | Клиенты |
| [bill/get\_not\_payed](#bill_get_not_payed) | просмотр всех неоплаченных счетов | Клиенты |
| [bill/get\_for\_period](#bill_get_for_period) | просмотр счетов за указанный период | Партнёры |
| [bill/change\_pay\_type](#bill_change_pay_type) | смена способа оплаты | Клиенты |
| [bill/delete](#bill_delete) | удаление неоплаченных счетов | Клиенты |
| Функции для работы с услугами | | |
| [service/nop](#service_nop) | тестовая функция | Клиенты |
| [service/get\_prices](#service_get_prices) | получение цен на активацию/продление услуг | Все, кроме индивидуальных тарифных планов [Рег. решения](https://www.reg.ru/vybor-tarifnogo-plana/) |
| [service/get\_servtype\_details](#service_get_servtype_details) | получение цен для услуги | Все |
| [service/create](#service_create) | заказ услуги | Все |
| [service/check\_create](#service_check_create) | валидация параметров заказа услуги | Все |
| [service/delete](#service_delete) | удаление услуги | Клиенты |
| [service/resend\_startup\_mail](#service_resend_startup_mail) | повторно выслать стартовое письмо активации услуги | Клиенты |
| [service/get\_info](#service_get_info) | получить информацию об услугах | Клиенты |
| [service/get\_list](#service_get_list) | получить список активных услуг | Клиенты |
| [service/get\_folders](#service_get_folders) | Получить список папок в которые входит сервис | Клиенты |
| [service/get\_details](#service_get_details) | получение параметров услуги | Клиенты |
| [service/service\_get\_details](#service_service_get_details) | получение параметров услуги | Клиенты |
| [service/get\_dedicated\_server\_list](#service_get_dedicated_server_list) | получение списка выделенных серверов доступных для заказа | Клиенты |
| [service/update](#service_update) | настройка услуги | Клиенты |
| [service/renew](#service_renew) | продление услуги | Клиенты |
| [service/get\_bills](#service_get_bills) | получение списка счетов, связанных с указанными услугами | Партнёры |
| [service/check\_readonly](#service_check_readonly) |  | Клиенты |
| [service/set\_autorenew\_flag](#service_set_autorenew_flag) | установить/снять флаг автопродления | Клиенты |
| [service/suspend](#service_suspend) | приостановить услугу (для домена - снятие делегирования) | Клиенты |
| [service/resume](#service_resume) | возобновить услугу (для домена - установить делегирование) | Клиенты |
| [service/get\_depreciated\_period](#service_get_depreciated_period) | расчитать дробное число периодов до истечения срока действия услуги | Клиенты |
| [service/upgrade](#service_upgrade) | произвести повышение подтипа (тарифа) услуги | Клиенты |
| [service/partcontrol\_grant](#service_partcontrol_grant) | предоставить право частичного управления услугой другому пользователю | Клиенты |
| [service/partcontrol\_revoke](#service_partcontrol_revoke) | отключить право частичного управления услугой | Клиенты |
| [service/resend\_mail](#service_resend_mail) | повторить email сообщение | Клиенты |
| [service/refill](#service_refill) | пополнить счёт услуги | Клиенты |
| [service/get\_balance](#service_get_balance) | Получить баланс внешней услуги | Клиенты |
| [service/seowizard\_manage\_link](#service_seowizard_manage_link) | получить ссылку на панель управления Seowizard'ом | Партнёры |
| [service/get\_websitebuilder\_link](#service_get_websitebuilder_link) |  | Партнёры |
| Функции для работы с доменами | | |
| [domain/nop](#domain_nop) | тестовая функция | Все |
| [domain/get\_prices](#domain_get_prices) | получение цен на регистрацию/продление доменов во всех доступных зонах | Все, кроме индивидуальных тарифных планов [Рег. решения](https://www.reg.ru/vybor-tarifnogo-plana/) |
| [domain/get\_suggest](#domain_get_suggest) | подбор имени домена | Партнёры |
| [domain/get\_premium\_prices](#domain_get_premium_prices) |  | Партнёры |
| [domain/get\_deleted](#domain_get_deleted) | список освобождённых доменов | Партнёры |
| [domain/check](#domain_check) | проверка доступности регистрации домена | Партнёры |
| [domain/create](#domain_create) | подать заявку на регистрацию домена | Клиенты |
| [domain/transfer](#domain_transfer) | подать заявку на перенос домена от другого регистратора | Клиенты |
| [domain/get\_transfer\_status](#get_transfer_status) |  | Клиенты |
| [domain/set\_new\_authinfo](#set_new_authinfo) |  | Клиенты |
| [domain/cancel\_transfer](#domain_cancel_transfer) | Отменить перенос домена | Партнёры |
| [domain/get\_rereg\_data](#domain_get_rereg_data) | получить список освобождающихся доменов с характеристиками, после регистрации или продления домены удаляются из списка | Партнёры |
| [domain/set\_rereg\_bids](#domain_set_rereg_bids) | сделать ставки на освобождающиеся домены | Клиенты |
| [domain/get\_user\_rereg\_bids](#domain_get_user_rereg_bids) | получить свои ставки на освобождающиеся домены, после регистрации или продления домены удаляются из списка | Клиенты |
| [domain/get\_docs\_upload\_uri](#domain_get_docs_upload_uri) | получение ссылки на закачивание документов из интернета для .RU/.SU/.РФ доменов | Клиенты |
| [domain/update\_contacts](#domain_update_contacts) | обновление контактных данных доменов | Клиенты |
| [domain/update\_private\_person\_flag](#domain_update_private_person_flag) | изменение флага «Private Person» скрытия/отображения контактных данных в whois | Клиенты |
| [domain/register\_ns](#domain_register_ns) | внести nameserver в NSI-registry | Все |
| [domain/delete\_ns](#domain_delete_ns) | удалить nameserver из NSI-registry | Все |
| [domain/get\_nss](#domain_get_nss) | Получение DNS серверов доменов | Клиенты |
| [domain/update\_nss](#domain_update_nss) | Изменение списка DNS серверов | Клиенты |
| [domain/delegate](#domain_delegate) | Установка флага делегирования домена | Все |
| [domain/undelegate](#domain_undelegate) | Снятие флага делегирования домена | Партнёры |
| [domain/transfer\_to\_another\_account](#domain_transfer_to_another_account) | Передача домена на другой аккаунт | Партнёры |
| [domain/look\_at\_entering\_list](#domain_look_at_entering_list) | Просмотр списка передаваемых доменов | Партнёры |
| [domain/accept\_or\_refuse\_entering\_list](#domain_accept_or_refuse_entering_list) | Принять или отклонить передаваемый домен | Партнёры |
| [domain/request\_to\_transfer](#domain_request_to_transfer) | Отправить заявку на перенос доменов на свой аккаунт | Партнёры c сервис планом "Партнёр 2" или выше |
| [domain/get\_tld\_info](#get_tld_info) |  | Клиенты |
| [domain/send\_email\_verification\_letter](#send_email_verification_letter) |  | Партнёры |
| [domain/download\_certificate](#download_certificate) |  | Партнёры |
| Функции для управления DNS-зоной | | |
| [zone/nop](#zone_nop) | тестовая функция | Клиенты |
| [zone/add\_alias](#zone_add_alias) | cвязать поддомен с IPv4-адресом | Клиенты |
| [zone/add\_aaaa](#zone_add_aaaa) | cвязать поддомен с IPv6-адресом | Клиенты |
| [zone/add\_caa](#zone_add_caa) | указать правила выпуска SSL сертификатов для поддомена | Клиенты |
| [zone/add\_cname](#zone_add_cname) | cвязать поддомен с именем другого домена | Клиенты |
| [zone/add\_https](#zone_add_https) | указать правила для выполнения HTTPS запроса к домену | Клиенты |
| [zone/add\_mx](#zone_add_mx) | указать почтовый сервер в виде доменного имени или IP-адреса, который будет принимать почту для вашего домена | Клиенты |
| [zone/add\_ns](#zone_add_ns) | передать управление поддоменами на другие DNS-сервера | Клиенты |
| [zone/add\_txt](#zone_add_txt) | добавить произвольную текстовую запись (TXT) для поддомена | Клиенты |
| [zone/add\_srv](#zone_add_srv) | добавить сервисную запись | Клиенты |
| [zone/get\_resource\_records](#zone_get_resource_records) | получение ресурсных записей зоны для каждого поддомена | Клиенты |
| [zone/update\_records](#zone_update_records) | добавить/удалить несколько ресурсных записей одним запросом | Партнёры |
| [zone/update\_soa](#zone_update_soa) | изменить время жизни кеша для зоны | Клиенты |
| [zone/tune\_forwarding](#zone_tune_forwarding) | настройка зоны для web-форвардинга | Клиенты |
| [zone/clear\_forwarding](#zone_clear_forwarding) | удалить записи настройки зоны для web-форвардинга | Клиенты |
| [zone/tune\_parking](#zone_tune_parking) | настройка зоны для парковки | Клиенты |
| [zone/clear\_parking](#zone_clear_parking) | удалить записи настройки зоны для парковки | Клиенты |
| [zone/remove\_record](#zone_remove_record) | удалить ресурсную запись | Клиенты |
| [zone/clear](#zone_clear) | удалить все ресурсные записи зоны | Клиенты |
| Функции для управления DNSSEC | | |
| [dnssec/nop](#dnssec_nop) | проверка доступности управления DNSSEC доменов | Клиенты |
| [dnssec/get\_status](#dnssec_get_status) | получение статуса DNSSEC домена | Клиенты |
| [dnssec/enable](#dnssec_enable) | включение DNSSEC для домена, использующего DNS сервера REG.RU | Клиенты |
| [dnssec/disable](#dnssec_disable) | выключение DNSSEC для домена, использующего DNS сервера REG.RU | Клиенты |
| [dnssec/renew\_ksk](#dnssec_renew_ksk) | обновление KSK ключа для домена, использующего DNS сервера REG.RU | Клиенты |
| [dnssec/renew\_zsk](#dnssec_renew_zsk) | обновление ZSK ключа для домена, использующего DNS сервера REG.RU | Клиенты |
| [dnssec/get\_records](#dnssec_get_records) | Получение списка DNSSEC записей домена | Клиенты |
| [dnssec/add\_keys](#dnssec_add_keys) | Передача информации о KSK ключах в родительскую зону | Клиенты |
| Функции для работы с хостингом | | |
| [hosting/nop](#hosting_nop) | проверка работоспособности API | Клиенты |
| [hosting/get\_os\_templates](#hosting_get_os_templates) | получение списка операционных систем | Все |
| Функции для работы с папками | | |
| [folder/nop](#folder_nop) | тестовая функция | Все |
| [folder/create](#folder_create) | создание папки | Все |
| [folder/remove](#folder_remove) | удаление папки | Все |
| [folder/rename](#folder_rename) | переименование папки | Все |
| [folder/get\_services](#folder_get_services) | выдать список услуг в папке | Все |
| [folder/add\_services](#folder_add_services) | добавление услуг в папку | Все |
| [folder/remove\_services](#folder_remove_services) | удаление услуг из папки | Все |
| [folder/replace\_services](#folder_replace_services) | перезаписывание услуг в папке | Все |
| [folder/move\_services](#folder_move_services) | перенос услуг из одной папки в другую | Все |
| Функции для работы с Магазином доменов | | |
| [shop/nop](#shop_nop) | тестовая функция | Клиенты |
| [shop/delete\_lot](#shop_delete_lot) | удаление лота | Все |
| [shop/get\_info](#shop_get_info) | получение информации о лоте | Все |
| [shop/get\_lot\_list](#shop_get_lot_list) | получение списка лотов | Все |
| [shop/get\_category\_list](#shop_get_category_list) | получение списка категорий лотов | Все |
| [shop/get\_suggested\_tags](#shop_get_suggested_tags) | получение списка ключевых слов лотов | Все |

# 4. Функции общего назначения

## 4.1. Функция: nop

##### Доступность:

Клиенты

##### Назначение

для тестирования, здесь — ничегонеделание + получение логина и идентификатора залогиненого пользователя

##### [Поля запроса:](#common_input_params)

нет

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| login | Имя пользователя, переданное в запросе как username. |
| user\_id | Идентификатор пользователя в системе. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/nop
password=test
username=test
```

##### Пример успешного ответа:

```
{
   "answer" : {
      "login" : "test",
      "user_id" : "0"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

пример ответа с ошибкой (неверное имя пользователя / пароль)

```
{
   "error_code" : "PASSWORD_AUTH_FAILED",
   "error_text" : "Username/password incorrect",
   "result" : "error"
}
```

Также см. [другие cтандартные коды ошибок](#common_errors)

## 4.2. Функция: reseller\_nop

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

полностью аналогична функции nop за исключением двух следующих пунктов

##### Режим доступа:

только защищённый HTTPS

##### Поля запроса:

нет

##### Поля ответа:

| Поле | Описание |
| --- | --- |
| login | Имя пользователя, переданное в запросе как username. |
| user\_id | Идентификатор пользователя в системе. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/reseller_nop
password=test
username=test
```

##### Возможные ошибки:

Пример ошибки, если обычный пользователь пытается получить доступ к функциям только для партнёров:

```
{
    error_text: 'Only resellers can access to this function',
    error_code: 'RESELLER_AUTH_FAILED',
    result: 'error'
}
```

Такая ошибка будет в случае попытки доступа не по HTTPS соединению:

```
{
    error_text: 'Access to api over non secure interface (http) prohibited!',
    error_code: 'HTTPS_ONLY',
    result: 'error'
}
```

## 4.3. Функция: get\_user\_id

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

для тестирования, возвращает идентификатор залогиненого пользователя

##### Поля запроса:

нет

##### Поля ответа:

| Поле | Описание |
| --- | --- |
| user\_id | Идентификатор пользователя в системе |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/get_user_id
password=test
username=test
```

##### Пример успешного ответа:

```
{
   "answer" : {
      "user_id" : "0"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 4.4. Функция: get\_service\_id

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получение id домена или услуги

##### Поля запроса:

[стандартные параметры идентификации сервисов](#common_service_identification_params)

##### Поля ответа:

| Поле | Описание |
| --- | --- |
| service\_id | идентификатор домена или услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/get_service_id
domain_name=qqq.ru
password=test
username=test
```

##### Пример успешного ответа:

```
{
   "answer" : {
      "service_id" : "123456"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 5. Функции для работы с учётной записью

## 5.1. Функция: user/nop

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

для тестирования доступности

##### [Поля запроса:](#common_input_params)

нет

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/nop
output_content_type=plain
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 5.2. Функция: user/create

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

регистрация нового пользователя

##### [Поля запроса:](#common_input_params)

##### Обязательные поля

###### Для регистрации с логином/паролем

| Параметр | Описание |
| --- | --- |
| user\_login | Логин нового пользователя в системе REG.RU, допустимые символы: латинские строчные буквы от "a" до "z", цифры "0" - "9", символы "-", "\_" |
| user\_password | Пароль нового пользователя, допускается использование любых латинских символов |
| user\_email | Адрес электронной почты нового пользователя |
| user\_ip | IP-адрес нового пользователя. Нужен для автоматического определения страницы проживания пользователя. |
| default\_country\_code | Код страны по умолчанию - используется, если не удалось определить страну автоматически по IP-адресу пользователя. |

###### Обязательные поля для упрощенной регистрации с указанием только email

Пароль будет сгенерирован автоматически и отправлен на указанный email.

| Параметр | Описание |
| --- | --- |
| email\_only | Флаг упрощенной регистрации с указанием только email-а. Установить в значение 1, если выбран вариант упрощенной регистрации нового пользователя |
| user\_email | Адрес электронной почты нового пользователя |
| user\_ip | IP-адрес нового пользователя. Нужен для автоматического определения страницы проживания пользователя. |
| default\_country\_code | Код страны по умолчанию - используется, если не удалось определить страну автоматически по IP-адресу пользователя. |

##### Необязательные поля — анкета пользователя

| Параметр | Описание |
| --- | --- |
| user\_first\_name | Имя контактного лица |
| user\_last\_name | Фамилия контактного лица |
| user\_company | Компания, в которой работает новый пользователь |
| user\_jabber\_id | Jabber ID нового пользователя |
| user\_icq | ICQ UIN нового пользователя |
| user\_phone | Номер телефона нового пользователя, телефон указывается с международном формате, например: +7.4951234567 |
| user\_fax | Номер факса нового пользователя, телефон указывается с международном формате, например: +7.4951234567 |
| user\_addr | Адрес нового пользователя: улица, дом, офис/квартира |
| user\_city | Адрес нового пользователя: город |
| user\_state | Адрес нового пользователя: область/край/штат |
| user\_postcode | Почтовый индекс нового пользователя |
| user\_wmid | Webmoney ID нового пользователя |
| user\_website | Веб-сайт нового пользователя |
| user\_language | Двухбуквенный код языка пользователя. Допустимы только 2 значения - ru/en |

##### Необязательные поля — другие параметры

| Параметр | Описание |
| --- | --- |
| user\_subsribe | Подписать пользователя на рассылку по электронной почте от REG.RU, допустимые значения 1 или 0 значение по умолчанию 0 |
| user\_mailnotify | Послать пользователю уведомление о регистрации, допустимые значения 1 или 0, значение по умолчанию 1 |
| set\_me\_as\_referrer | Сделать регистрируемого пользователя рефералом регистратора, допустимые значения 1 или 0, значение по умолчанию 0 |
| check\_only | Не регистрировать пользователя, только проверить контакты, допустимые значения 1 или 0, значение по умолчанию 0, при check\_only=1 в ответе будет user\_id=777 |
| white\_list\_ips | Один или несколько диапазонов IP-адресов, с которых будет доступна работа по API для данного пользователя. Примеры: одиночный IP-адрес: 192.168.0.1, 32-битная CIDR-нотация: 192.168.0.1/32, с маской подсети: 192.168.0.1/255.255.255.255, неполный IP-адрес (пустые биты являются адресом сети): 192.168.  Примеры передачи списка:  PLAIN: white\_list\_ips=192.168.0.1&white\_list\_ips=192.168.0.2&white\_list\_ips=192.168.0.3  JSON: "white\_list\_ips":["192.168.0.1","192.168.0.2","192.168.0.3"]  XML: <white\_list\_ips>192.168.0.1</white\_list\_ips><white\_list\_ips>192.168.0.2</white\_list\_ips> |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| user\_id | идентификатор только что созданного пользователя |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/create
output_content_type=plain
password=test
set_me_as_referrer=1
user_country_code=RU
user_email=test@example.com
user_login=othertest
user_mailnotify=0
user_password=xxxxx
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "user_id" : "777"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| USER\_LOGIN\_NOT\_UNIQUE | This login already is busy | Этот логин уже занят |

Cм. [cтандартные коды ошибок](#common_errors)

## 5.3. Функция: user/get\_statistics

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получение статистики по пользователю

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| date\_from | задать начальную дату для параметров, необязательный параметр |
| date\_till | задать конечную дату для параметров, необязательный параметр |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| costs\_for\_period | потрачено средств за указанный период |
| active\_domains\_cnt | кол-во активных доменов, в т.ч. полученных в частичное управление |
| active\_domains\_get\_ctrl\_cnt | кол-во доменов, полученных в частичное управление |
| renew\_domains\_cnt | кол-во доменов, требующих продления |
| renew\_domains\_get\_ctrl\_cnt | кол-во доменов, требующих продления из полученных в частичное управление |
| trans\_in\_domains\_cnt | кол-во доменов, ожидающих переноса в REG.RU, если такие есть |
| undelegated\_domains\_cnt | кол-во неделегированных доменов |
| reg\_domains\_cnt | кол-во зарегистрированных доменов за период |
| domain\_folders\_cnt | кол-во доменных папок |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/get_statistics
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "active_domains_cnt" : "5",
      "active_domains_get_ctrl_cnt" : "3",
      "domain_folders_cnt" : "2",
      "renew_domains_cnt" : "4",
      "renew_domains_get_ctrl_cnt" : "1",
      "undelegated_domains_cnt" : "6"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 5.4. Функция: user/get\_balance

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

просмотр суммы на счёте

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| currency | указать валюту в которой выводится сумма, пересчёт из рублей производится по текущему курсу; доступные варианты: RUR, USD, EUR, UAH; значение по умолчанию — RUR |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| currency | валюта в которую пересчитаны суммы в момент запроса |
| prepay | сумма предоплаты, находящаяся на счёте |
| blocked | сумма заблокированная на счёте, например, на время торгов для участия в акуционе доменов, отображается при ненулевом значении |
| credit | сумма предоставляемого кредита, доступно для Партнёров |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/get_balance
currency=RUR
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "credit" : "10",
      "currency" : "RUR",
      "prepay" : "1000"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 5.5. Функция: user/set\_reseller\_url

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Установка URL для редиректа из внешней услуги

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| servtype | тип услуги, для которой будет выставлен URL. Доступные типы:  srv\_seowizard |
| url\_type | тип URL для редиректа. Доступные типы:  refill  refund |
| url | URL для редиректа. Может содержать следующие токены (вместо токена будет подставлено соответствующее значение):  <service\_id> — идентификатор услуги, из которой вызывается редирект  <email> — Email, использованный при создании услуги (если использовался) |

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/set_reseller_url
password=test
servtype=srv_seowizard
url=http://test2.com
url_type=refill
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 5.6. Функция: user/get\_reseller\_url

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение URL для редиректа из внешней услуги.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| servtype | тип услуги, для которой нужно получить URL. Доступные типы:  srv\_seowizard |
| url\_type | тип URL для редиректа. Доступные типы:  refill  refund |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| url | URL для редиректа. Может содержать следующие токены (вместо токена будет подставлено соответствующее значение):  <service\_id> — идентификатор услуги, из которой вызывается редирект  <email> — Email, использованный при создании услуги (если использовался) |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/user/get_reseller_url
password=test
servtype=srv_seowizard
url_type=refill
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "url" : "http://test2.com"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 6. Функции для работы cо счетами (категория bill)

## 6.1. Функция: bill/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка счетов:

Да

##### Назначение:

для тестирования

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| bill\_id | номер счёта при запросе с одиночным счётом |
| bills | список номеров счетов |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bills | список счетов |
| bill\_id | номер счёта |
| pay\_status | статус оплаченности счёта, варианты статусов см. ниже |

##### Примеры запросов:

```
https://api.reg.ru/api/regru2/bill/nop
bill_id=12345
output_content_type=plain
password=test
username=test
```

```
https://api.reg.ru/api/regru2/bill/nop
input_data={"bills":["12345","12346"],"output_content_type":"plain"}
input_format=json
password=test
username=test
```

```
https://api.reg.ru/api/regru2/bill/nop
input_data={"bills":[{"bill_id":"12345"},{"bill_id":"12346"}],"output_content_type":"plain"}
input_format=json
password=test
username=test
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 6.2. Функция: bill/get\_not\_payed

##### Доступность:

Клиенты

##### Поддержка обработки списка счетов:

Да

##### Назначение:

получение списка неоплаченных счетов

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| limit | количество позиций счётов, допустимых к выводу за 1 раз, значение по умолчанию 100, максимальное значение 1024 |
| offset | смещение от начальной позиции, если количество счетов превышает указанный limit |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bills | список счетов |
| bill\_id | номер счёта |
| bill\_date | дата создания счёта |
| currency | валюта (RUR - рубли, USD - доллары США, EUR - евро, UAH - украинские гривны) |
| payment | сумма счёта без процентов платёжных систем в рублях |
| total\_payment | полная сумма к оплате в указанной Вами валюте, состоит из суммы цен услуг счёта, процентов за перевод денег в указанной системе оплаты, и процентов за конвертацию указанной валюты в рубли, если платёж осуществляется не в рублях |
| pay\_type | способ платежа, варианты оплаты: prepay – использование предоплаты, bank – банковский перевод, pbank – безналичный перевод через банк, yacard – ЮMoney |
| pay\_status | статус оплаченности счёта, здесь будут только notpayed — не оплачено |
| items | состав счёта |
| itemtype | тип позиции счёта: prepayment - предоплата, service - заказ услуги |
| dname | имя домена сервиса, если применимо |
| servtype | тип сервиса |
| service\_id | id сервиса |
| action | заказ нового или продление уже имеющегося сервиса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/bill/get_not_payed
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "bills" : [
         {
            "payment" : "100",
            "pay_status" : "notpayed",
            "currency" : "RUR",
            "bill_date" : "1917-10-26",
            "pay_type" : "bank",
            "pay_date" : null,
            "bill_id" : "12345",
            "orig_payment" : "100",
            "items" : [
               {
                  "dname" : "october.org",
                  "itemtype" : "service",
                  "action" : "new",
                  "service_id" : "12345",
                  "servtype" : "domain"
               },
               {
                  "dname" : "october.org",
                  "itemtype" : "service",
                  "action" : "new",
                  "service_id" : "12346",
                  "servtype" : "srv_certificate"
               }
            ]
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 6.3. Функция: bill/get\_for\_period

##### Доступность:

Партнёры

##### Поддержка обработки списка счетов:

Да

##### Назначение:

получение списка счетов за указанный период

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| start\_date | начальная дата периода для запроса счетов в ISO формате, обязательное поле |
| end\_date | конечное дата периода для запроса счетов в ISO формате, обязательное поле |
| pay\_type | способ оплаты счёта, допустимые варианты см. в описании полей ответа |
| limit | количество позиций счётов, допустимых к выводу за 1 раз, значение по умолчанию 100, максимальное значение 1024 |
| offset | смещение от начальной позиции, если количество счетов превышает указанный limit |
| all | так же показывать неактивные счета, т.е. у которых истекли сроки действия заказываемых сервисов или по которым был осуществлён возврат средств из-за невозможности выполнения заказа |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bills | список счетов |
| bill\_id | номер счёта |
| bill\_date | дата создания счёта |
| currency | валюта (RUR - рубли, USD - доллары США, EUR - евро, UAH - украинские гривны) |
| payment | сумма счёта без процентов платёжных систем в рублях |
| total\_payment | полная сумма к оплате в указанной Вами валюте, состоит из суммы цен услуг счёта, процентов за перевод денег в указанной системе оплаты, и процентов за конвертацию указанной валюты в рубли, если платёж осуществляется не в рублях |
| pay\_type | способ платежа, варианты оплаты: prepay – использование предоплаты, bank – банковский перевод, pbank – безналичный перевод через банк, yacard – ЮMoney |
| pay\_status | статус оплаченности счёта, возможные варианты: notpayed – счёт не оплачен, confirmed – оплата подтверждена, но деньги ещё не получены (например долгий перевод через банк), payed – оплачено, cancelled – оплата отменена на стороне платёжной системы |
| items | состав счёта |
| itemtype | тип позиции счёта: prepayment - предоплата, service - заказ услуги |
| dname | имя домена сервиса, если применимо |
| servtype | тип сервиса |
| service\_id | id сервиса |
| action | заказ нового или продление уже имеющегося сервиса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/bill/get_for_period
end_date=1917-11-07
output_content_type=plain
password=test
start_date=1917-10-26
username=test
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 6.4. Функция: bill/change\_pay\_type

##### Доступность:

Клиенты

##### Поддержка обработки списка счетов:

Да

##### Назначение:

изменение способа оплаты счёта, для некоторых способов возможно выставление счёта в указанной системе оплаты, для prepay оплата производится сразу

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| bill\_id | номер счёта при запросе с одиночным счётом |
| bills | список номеров счетов |
| pay\_type | новый тип платежа, обязательное поле, возможные варианты:  prepay — предоплата, оплата производится сразу;  yamoney — ЮMoney;  bank — оплата через банк |
| currency | валюта, обязательное поле, для ЮMoney доступно только RUR, для bank и prepay — RUR и USD |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bills | список счетов |
| bill\_id | номер счёта |
| currency | валюта (RUR - рубли, USD - доллары США) |
| payment | сумма счёта без процентов платёжных систем в рублях |
| total\_payment | полная сумма к оплате в указанной Вами валюте, состоит из суммы цен услуг счёта, процентов за перевод денег в указанной системе оплаты, и процентов за конвертацию указанной валюты в рубли, если платёж осуществляется не в рублях |
| pay\_type | способ платежа |
| pay\_status | статус оплаченности счёта: notpayed – счёт не оплачен, payed – оплачено, cancelled – оплата отменена на стороне платёжной системы |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/bill/change_pay_type
input_data={"bills":["123456"],"pay_type":"prepay","currency":"RUR","output_content_type":"plain"}
input_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "bills" : [
         {
            "bill_id" : "123456",
            "currency" : "RUR",
            "old_pay_type" : "bank",
            "pay_status" : "payed",
            "pay_type" : "prepay",
            "payment" : "100",
            "result" : "success",
            "total_payment" : "100"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 6.5. Функция: bill/delete

##### Доступность:

Клиенты

##### Поддержка обработки списка счетов:

Да

##### Назначение:

Удаление неоплаченных счетов

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| bill\_id | номер счёта при запросе с одиночным счётом |
| bills | список номеров счетов |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bills | список счетов |
| bill\_id | номер счёта |
| status | статус счёта, выставляется только реальных счетов, deleted для удалённого счёта, active для уже оплаченного счёта не подлежащего удалению |
| pay\_status | статус оплаченности счёта, варианты статусов см. выше |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/bill/delete
bill_id=12345
output_content_type=plain
password=test
username=test
```

```
https://api.reg.ru/api/regru2/bill/delete
input_data={"bills":["12345","12346","12347"],"output_content_type":"plain"}
input_format=json
```

```
https://api.reg.ru/api/regru2/bill/delete
input_data={"bills":[{"bill_id":"12345"},{"bill_id":"12346"},{"bill_id":"12347"}],"output_content_type":"plain"}
input_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "bills" : [
         {
            "bill_id" : "12345",
            "pay_status" : "notpayed",
            "result" : "success",
            "status" : "deleted"
         },
         {
            "bill_id" : "12346",
            "error_code" : "BILL_CAN_NOT_REMOVED",
            "pay_status" : "payed",
            "status" : "active"
         },
         {
            "bill_id" : "12347",
            "error_code" : "BILL_ID_NOT_FOUND"
         }
      ]
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 7. Функции для работы с услугами

## 7.1. Функция: service/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

для тестирования, позволяет проверить доступность списка услуг и получить их id

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| service\_id | идентификатор услуги, если переданы dname+servtype |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/nop
domain_name=test.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.2. Функция: service/get\_prices

##### Доступность:

Все, кроме индивидуальных тарифных планов [Рег. решения](https://www.reg.ru/vybor-tarifnogo-plana/)

##### Назначение:

Получение цен на регистрацию/продление услуг.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| [параметры для аутентификации](#common_auth_params) | Используйте аутентификацию для получения партнёрских цен. |
| show\_renew\_data | Флаг возврата цен для продления (1/0). Необязательное поле. |
| currency | Идентификатор валюты, в которой будут возвращаться цены (rur, uah, usd, eur). Необязательное поле, по умолчанию цены указываются в рублях. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_prices
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
  "answer" => {
      "currency" => "RUR",
      "pricegroup" => "0",
      "show_renew_data" => "0",
      "prices" => [
         {
            "servortld" => "srv_addip",
            "unit" => "month",
            "subtype" => "",
            "price" => "30"
         },
         {
            "servortld" => "srv_addip",
            "unit" => "month",
            "subtype" => "custom_net",
            "price" => "60"
         },
         ...
      ]
      "result" => "success"
  }
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.3. Функция: service/get\_servtype\_details

##### Доступность:

Все

##### Назначение:

Получение цены и общих данных для услуги.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| servtype | Вид услуги:  srv\_webfwd — «Web-форвардинг»,  srv\_parking — «Парковка домена»,  srv\_dns\_both — «Поддержка DNS»,  srv\_hosting\_ispmgr — «Хостинг»,  srv\_certificate — «Сертификат на домен»,  srv\_voucher — «Свидетельство на домен»,  srv\_kvm — «KVM доступ». |
| subtype | Подтип услуги |
| unroll\_prices | Показывать цены в развернутом виде |
| only\_actual | Если задан - показывать только актуальные тарифы |
| show\_hidden | Включая архивные тарифы |

Примечание:

Чтобы получить цены для нескольких видов услуг, можно указать их в поле `servtype` через запятую или передать в запросе несколько полей `servtype`. В этом случае поле `subtype` игнорируется.

##### [Поля ответа:](#common_response_parameters)

Список подтипов услуги. Каждый элемент списка содержит поля:

| Поле | Описание |
| --- | --- |
| servtype | Вид услуги |
| subtype | Подтип услуги |
| unit | Единица измерения для периода "YEAR" или "MONTH" |
| extparams | Дополнительные параметры |
| is\_renewable | 1 - возможно продление  0 - услуга без продления |
| commonname | Различные форматы описания |

Дополнительные поля при вызове функции с отсутствующим или нулевым параметром `unroll_prices`:

| Поле | Описание |
| --- | --- |
| periods\_new | Диапазон возможных сроков регистрации |
| periods\_renew | Диапазон возможных сроков продления |
| price\_new | Цена заказа сервиса |
| price\_renew | Цена продления сервиса |

Дополнительные поля при вызове функции с параметром `unroll_prices=1`:

| Поле | Описание |
| --- | --- |
| prices\_new | Список периодов и цен для заказа услуги |
| prices\_renew | Список периодов и цен для продления услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_servtype_details
output_content_type=plain
password=test
servtype=srv_hosting_ispmgr
username=test
```

##### Пример ответа:

```
{
   "answer" : [
      {
         "commonname" : "Web-forwarding",
         "extparams" : {},
         "is_renewable" : "1",
         "periods_new" : "1-10",
         "periods_renew" : "1-10",
         "price_new" : "120.00",
         "price_renew" : "120.00",
         "servtype" : "srv_webfwd",
         "subtype" : "",
         "unit" : "year"
      }
   ],
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.4. Функция: service/create

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

заказ новой услуги.

##### [Поля запроса:](#common_input_params)

##### Общие

| Параметр | Описание |
| --- | --- |
| domain\_name | Имя домена, для которого заказывается услуга. |
| servtype | Вид заказываемой услуги:  srv\_webfwd — «Web-форвардинг»,  srv\_parking — «Парковка домена»,  srv\_dns\_both — «Поддержка DNS»,  srv\_hosting\_ispmgr — «ISPmanager хостинг»,  srv\_hosting\_cpanel — «CPanel хостинг»,  srv\_hosting\_plesk — «PLesk хостинг»,  srv\_ssl\_certificate — «SSL сертификат»,  srv\_certificate — «Сертификат на домен»,  srv\_voucher — «Свидетельство на домен»,  srv\_license\_isp — «Лицензия ISP Manager»,  srv\_addip — «Дополнительный IP»,  srv\_antispam — «Расширенная защита от спама»,  srv\_dedicated — «Выделенный сервер»,  srv\_kvm — «KVM доступ»,  srv\_seowizard — «Автоматическое SEO-продвижение»,  srv\_websitebuilder — «Конструктор сайтов REG.RU» |
| period | Срок, на который заказывается услуга, единица измерения (год или месяц) зависит от вида заказываемой услуги.  Значения единиц измерения для различных услуг Вы можете получить с помощью функции [service/get\_servtype\_details](#service_get_servtype_details). |
| user\_servid | ID домена, задаваемый пользователем. Допустимые символы: цифры 0..9 и латинские буквы a..f, длина поля 32 символа. Автоматически идентификатор не создаётся, т.е. если он не был задан при создании услуги, то поле остаётся пустым. Необязательное поле. |

##### Параметры оплаты (необязательны)

| Параметр | Описание |
| --- | --- |
| pay\_type  ok\_if\_no\_money | См. [Общие параметры оплаты](#common_payment_params). |

##### Прочие общие параметры (необязательны)

| Параметр | Описание |
| --- | --- |
| folder\_name  или folder\_id | Задает название папки, куда будут добавлены услуги.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| no\_new\_folder | Если указано имя несуществующей папки:  0 — (по-умолчанию) — Создавать новую папку.  1 — Не создавать папку, возвращать код ошибки.  Необязательное поле. |
| comment | Комментарий - любая строка описывающая заказ. Необязательное поле. |
| admin\_comment | Комментарий для администраторов — любая строка описывающая заказ. Необязательное поле. |

##### ISPmanager хостинг (srv\_hosting\_ispmgr)

| Параметр | Описание |
| --- | --- |
| plan (deprecated) | Тарифный план, сейчас доступны: "Host-0-0420", "Host-1-0420", "Host-3-0420", "Host-A-0420", "Host-B-0420", "Host-Lite-0420". Для указания тарифного плана рекомендуется использовать параметр "subtype" |
| subtype | Тарифный план, сейчас доступны: "Host-0-0420", "Host-1-0420", "Host-3-0420", "Host-A-0420", "Host-B-0420", "Host-Lite-0420". Для указания тарифного плана рекомендуется использовать параметр "subtype" |
| contype | Тип контактных данных. Принимает значение "hosting\_pp" при регистрации сервера на данные физического лица и значение "hosting\_org" при регистрации сервера на данные юридического лица. |
| email | e-mail-адрес для создаваемого хостинг-аккаунта. |
| phone | Телефон. Необязательное поле |
| country | Двухбуквенный ISO-код страны, в которой зарегистрировано(а) физическое лицо (организация). |
| person\_r | Указывается при использовании параметра "contype" со значением "hosting\_pp". Фамилия, имя и отчество администратора сервера на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации). Сокращения длиной в идин символ, в этом поле приняты не будут. Например сокращение вида Vasily N Pupkin - вызовет ошибку для данного поля.  Примеры правильного заполнения:    *Пример1: Пупкин Василий Николаевич*  *Пример2: John Smith* |
| passport | Указывается при использовании параметра "contype" со значением "hosting\_pp". Серия и номер паспорта, а также наименование органа, выдавшего паспорт, и дата выдачи (в указанной последовательности, с разделением пробелами). В написании римских цифр допустимо использование только латинских букв. Дата записывается в формате ДД.ММ.ГГГГ. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются. В случае использования документа, отличного от паспорта (допустимо ТОЛЬКО для нерезидентов России), в начале строки указывается наименование вида документа. Запись может быть многострочной.  *Пример: 34 02 651241 выдан 48 о/м г.Москвы 26.12.1990* |
| pp\_code | Указывается при использовании параметра "contype" со значением "hosting\_pp". Идентификационный номер налогоплательщика (ИНН). Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  Необязательное поле. *Пример: 7701107259* |
| org\_r | Указывается при использовании параметра "contype" со значением "hosting\_org". Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке). Запись может быть многострочной.  *Пример1: Урюпинский государственный университет\nимени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | Указывается при использовании параметра "contype" со значением "hosting\_org". Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |

##### Хостинг Cpanel (srv\_hosting\_cpanel)

| Параметр | Описание |
| --- | --- |
| subtype | Тарифный план, сейчас доступны: "cPanel-L-0222", "cPanel-M-0222", "cPanel-S-0222", "cPanel-XL-0222". |
| contype | Тип контактных данных. Принимает значение "hosting\_pp" при регистрации сервера на данные физического лица и значение "hosting\_org" при регистрации сервера на данные юридического лица. |
| email | e-mail-адрес для создаваемого хостинг-аккаунта. |
| phone | Телефон. Необязательное поле |
| country | Двухбуквенный ISO-код страны, в которой зарегистрировано(а) физическое лицо (организация). |
| person\_r | Указывается при использовании параметра "contype" со значением "hosting\_pp". Фамилия, имя и отчество администратора сервера на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации). Сокращения длиной в идин символ, в этом поле приняты не будут. Например сокращение вида Vasily N Pupkin - вызовет ошибку для данного поля.  Примеры правильного заполнения:    *Пример1: Пупкин Василий Николаевич*  *Пример2: John Smith* |
| passport | Указывается при использовании параметра "contype" со значением "hosting\_pp". Серия и номер паспорта, а также наименование органа, выдавшего паспорт, и дата выдачи (в указанной последовательности, с разделением пробелами). В написании римских цифр допустимо использование только латинских букв. Дата записывается в формате ДД.ММ.ГГГГ. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются. В случае использования документа, отличного от паспорта (допустимо ТОЛЬКО для нерезидентов России), в начале строки указывается наименование вида документа. Запись может быть многострочной.  *Пример: 34 02 651241 выдан 48 о/м г.Москвы 26.12.1990* |
| pp\_code | Указывается при использовании параметра "contype" со значением "hosting\_pp". Идентификационный номер налогоплательщика (ИНН). Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  Необязательное поле. *Пример: 7701107259* |
| org\_r | Указывается при использовании параметра "contype" со значением "hosting\_org". Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке). Запись может быть многострочной.  *Пример1: Урюпинский государственный университет\nимени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | Указывается при использовании параметра "contype" со значением "hosting\_org". Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |

##### Хостинг Plesk (srv\_hosting\_plesk)

| Параметр | Описание |
| --- | --- |
| subtype | Тарифный план, сейчас доступны: "Plesk-L-0322", "Plesk-M-0322", "Plesk-S-0322", "Plesk-XL-0322". |
| contype | Тип контактных данных. Принимает значение "hosting\_pp" при регистрации сервера на данные физического лица и значение "hosting\_org" при регистрации сервера на данные юридического лица. |
| email | e-mail-адрес для создаваемого хостинг-аккаунта. |
| phone | Телефон. Необязательное поле |
| country | Двухбуквенный ISO-код страны, в которой зарегистрировано(а) физическое лицо (организация). |
| person\_r | Указывается при использовании параметра "contype" со значением "pp". Фамилия, имя и отчество администратора сервера на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации). Сокращения длиной в идин символ, в этом поле приняты не будут. Например сокращение вида Vasily N Pupkin - вызовет ошибку для данного поля.  Примеры правильного заполнения:    *Пример1: Пупкин Василий Николаевич*  *Пример2: John Smith* |
| passport | Указывается при использовании параметра "contype" со значением "pp". Серия и номер паспорта, а также наименование органа, выдавшего паспорт, и дата выдачи (в указанной последовательности, с разделением пробелами). В написании римских цифр допустимо использование только латинских букв. Дата записывается в формате ДД.ММ.ГГГГ. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются. В случае использования документа, отличного от паспорта (допустимо ТОЛЬКО для нерезидентов России), в начале строки указывается наименование вида документа. Запись может быть многострочной.  *Пример: 34 02 651241 выдан 48 о/м г.Москвы 26.12.1990* |
| pp\_code | Указывается при использовании параметра "contype" со значением "hosting\_pp". Идентификационный номер налогоплательщика (ИНН). Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  Необязательное поле. *Пример: 7701107259* |
| org\_r | Указывается при использовании параметра "contype" со значением "hosting\_org". Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке). Запись может быть многострочной.  *Пример1: Урюпинский государственный университет\nимени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | Указывается при использовании параметра "contype" со значением "org". Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |

##### Парковка (srv\_parking)

| Параметр | Описание |
| --- | --- |
| title | Заголовок страницы. Просто элемент оформления шаблона. |
| content | HTML-код страницы. |
| counter\_html\_code | HTML-код счётчиков (опционально). |
| template\_name | Идентификатор шаблона. Доступные идентификаторы:  private\_property  «Частная собственность»  under\_construction  «На сайте идут строительные работы»  for\_sale  «Домен выставлен на продажу»  for\_rent  «Домен сдается в аренду»  for\_you\_gift  «Для Вас подарок»  happy\_new\_year  «С Новым годом!»  happy\_birthday1  «С Днем рождения!» (вариант 1)  happy\_birthday2  «С Днем рождения!» (вариант 2)  love\_happiness  «Любви и счастья!»  empty  Пустая страница  raw\_html  Ваш HTML код (только для платной парковки, без стандарного рекламного блока) |
| html\_title | HTML мета-тег «Title». |
| html\_description | HTML мета-тег «Description». |
| html\_keywords | HTML мета-тег «Keywords». |
| opt\_user\_contacts | Отобразить информационный блок с контактами Администратора домена. |
| opt\_feedback\_link | Отобразить информационный блок со ссылкой "Связь с Администратором домена" (ссылка на Whois по домену).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_domain\_shop\_link | Отобразить информационный блок со ссылкой на лот в Магазине доменов (при условии, что домен выставлен на продажу в Магазине доменов).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_whois\_link | Отобразить информационный блок со ссылкой на историю Whois домена.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_se\_link | Отобразить информационный блок со cсылками на домен в поисковых системах.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_indexed\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_blogs\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| subtype | Необязательный. Для бесплатной парковки следует указать значение free |

##### Пример запроса для парковки:

```
https://api.reg.ru/api/regru2/service/create
dname=qqq.ru
output_content_type=plain
password=test
servtype=srv_parking
username=test
```

##### Пример ответ:

```
{
    'result' => 'success';
    'answer' => {
        'descr' => 'service srv_parking is ordered for domain qqq.ru',
        'payment' => '100',
        'pay_notes' => 'Amount successfully charged',
        'bill_id' => '1234',
        'service_id' => '987654'
     }
}
```

##### Пример запроса для бесплатной парковки:

```
https://api.reg.ru/api/regru2/service/create
dname=qqq.ru
output_content_type=plain
password=test
servtype=srv_parking
subtype=free
username=test
```

##### Пример ответа:

```
{
    'result' => 'success';
    'answer' => {
        'descr' => 'service srv_parking is ordered for domain qqq.ru',
        'payment' => '0',
        'pay_notes' => 'Amount successfully charged',
        'bill_id' => '1234',
        'service_id' => '987654'
     }
}
```

##### Сертификат SSL (srv\_ssl\_certificate)

| Параметр | Описание |
| --- | --- |
| **обязательные:**  org\_org\_name, org\_address, org\_city, org\_state, org\_postal\_code, org\_country, org\_phone    **необязательные( кроме GlobalSign):**  org\_first\_name, org\_last\_name, org\_fax | Название, имя, фамилия, адрес, город, штат(провинция), почтовый индекс, страна, телефон, факс организации.    Используются для всех SSL сертификатов компаний Thawte, Comodo, VeriSign, а так-же для GeoTrust SSL сертификатов: True BusinessID, True BusinessID Wildcard, True BusinessID with EV.    Для GlobalSign обязательными являютcя org\_org\_name, org\_address, org\_city, org\_state, org\_postal\_code, org\_country, org\_phone, org\_email, org\_first\_name, org\_last\_name  используются для заказа сертификатов gs\_organizationssl, gs\_organizationssl\_wildcard |
| **обязательные:**  admin\_first\_name, admin\_last\_name, admin\_title, admin\_address, admin\_city, admin\_state, admin\_postal\_code, admin\_country, admin\_phone, admin\_email,    **необязательные:**  admin\_fax, admin\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес администратора.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust, Trustwave и VeriSign, а так-же для Comodo EV SSL сертификата.    Для GlobalSign являютcя обязательными для всех видов сертификатов admin\_city, admin\_country, admin\_email, admin\_first\_name, admin\_last\_name, admin\_phone, admin\_state |
| **обязательные:**  billing\_first\_name, billing\_last\_name, billing\_title, billing\_address, billing\_city, billing\_state, billing\_postal\_code, billing\_country, billing\_phone, billing\_email,    **необязательные:**  billing\_fax, billing\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес финансового менеджера.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust и VeriSign. |
| **обязательные:**  tech\_first\_name, tech\_last\_name, tech\_title, tech\_address, tech\_city, tech\_state, tech\_postal\_code, tech\_country, tech\_phone, tech\_email,    **необязательные:**  tech\_fax, tech\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес технического специалиста.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust и VeriSign. |
| **обязательные:**  signer\_first\_name, signer\_last\_name, signer\_title, signer\_address, signer\_city, signer\_state, signer\_postal\_code, signer\_country, signer\_phone, signer\_email,    **необязательные:**  signer\_fax, signer\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес ответственного за подписку SSL сертиификата.    Используются для Comodo EV SSL сертификата. |
| **обязательные:**  evorg\_email, evorg\_first\_name, evorg\_last\_name, evorg\_phone, evorg\_org\_name | Адрес, город, страна, e-mail, имя, фамилия, телефон, почтовый индекс, штат  код категории бизнеса принимает значения( PO, GE, BE).  BE:BusinessEntity  PO:Private Organization  GE:Government Entity    Данные организации  Используются для GlobalSign EV SSL сертификата. |
| **обязательные:**  evsigner\_email, evsigner\_first\_name, evsigner\_last\_name, evsigner\_phone, evsigner\_org\_name | e-mail, имя, фамилия, телефон, имя организации    Данные ответственного за подписку SSL сертификата.  Используются для GlobalSign EV SSL сертификата. |
| approver\_email | E-mail адрес для подтверждения сертификата. |
| server\_type | Программное обеспечение. Возможные значения:  Для Comodo SSL сертификатов:  apachessl, citrix, domino, ensim, hsphere, iis4, iis6, iis7, iplanet, javawebserver, netscape, ibmhttp, novell, oracle, other, plesk, redhat, sap, tomcat, webstar, whmcpanel    Для Thawte, GeoTrust и VeriSign SSL сертификатов:  apache2 apacheopenssl apacheraven apachessl apachessleay c2net cobaltseries cobaltraq3 cobaltraq2 cpanel domino dominogo4626 dominogo4625 ensim hsphere iis iis4 iis5 iplanet ipswitch netscape ibmhttp other plesk tomcat weblogic website webstar webstar4 zeusv3    Для TrustWave SSL Сертификатов данный параметр не используется.    Для GlobalSign только два значения: iis, not\_iis |
| csr\_string | Закодированный CSR включая маркеры начала и окончания. |
| subtype | Вид сертификата. Возможные значения:    **GlobalSign:**  gs\_alphassl\_wildcard, gs\_domainssl\_wildcard,  gs\_organizationssl, gs\_extendedssl, gs\_organizationssl\_wildcard,  gs\_alphassl\_dns, gs\_domainssl\_dns, gs\_alphassl\_dns\_wildcard,  gs\_domainssl\_dns\_wildcard  **gs\_domainssl\_free**( для заказа достаточно двух параметров service\_id и dname)    **Thawte:**  ssl123, sgcsuper\_certs, sslwebserver, sslwebserver\_wildcard, sslwebserver\_ev    **Symantec:**  securesite, securesite\_pro, securesite\_ev, securesite\_pro\_ev    **GeoTrust:**  quickssl, quickssl\_premium, truebizid, truebizid\_wildcard, truebizid\_ev    **TrustWave:**  trustwave\_dv, trustwave\_ev, trustwave\_premiumssl, trustwave\_premiumssl\_wildcard    **Comodo:**  comodo\_ev, comodo\_instantssl, comodo\_premiumssl, comodo\_premiumssl\_wildcard, comodo\_ssl, comodo\_wildcard |
| dname | **Обязательное поле для заказа субсервиса бесплатного сертификата.**  Имя домена для сертификата. IDN запрещены. |
| service\_id | **Обязательное поле для заказа субсервиса бесплатного сертификата.**  id родительского сервиса(хостинг или домен), к которому будет прикрепляться сертификат. |

##### Сертификат на домен (srv\_certificate)

##### Свидетельство на домен (srv\_voucher)

| Параметр | Описание |
| --- | --- |
| subtype | Подтип услуги. Например «payed» для «srv\_certificate». subtype «digital» доступен только для услуги «srv\_voucher», при этом все остальные параметры ниже становятся необязательны |
| obtain\_cert | способ получения сертификата:  in\_office — В офисе REG.RU  дополнительные параметры: office, phone, remark  free\_mail — Почтой в любой город Российской Федерации (доставка бесплатная)  дополнительные параметры: postcode, name, addr  paid\_mail — Почтой в любой другой город мира (доставка платная)  дополнительные параметры: postcode, name, addr, city, country\_code, state |
| office | Допустимые значения: moscow, samara, kiev, piter |
| phone | телефон в международном формате: знак "+", код страны, номер телефона. |
| remark | Примечание |
| p\_postcode | Почтовый индекс для бесплатной отправки сертификата по России |
| p\_addr | Адрес в России |
| p\_name | Фамилия, Имя, Отчество по-русски |
| a\_postcode | Почтовый индекс для международного письма |
| a\_addr | Почтовый адрес для международного (только латиницей) |
| a\_name | Полное имя для международного письма (только латиницей) |
| a\_state | Область, штат |
| a\_city | Город для международной отправки сертификата |
| a\_country\_code | Код страны (например UK) |

##### Лицензия ISP Manager (srv\_license\_isp)

| Параметр | Описание |
| --- | --- |
| uplink\_service\_id | ID родительской услуги, к которой заказывается лицензия ISP Manager. Заказ лицензии возможен только для Dedicated (srv\_dedicated). |
| subtype | Тип лицензии, доступны: "lite5", "business".  Необязательный параметр, значение по умолчанию: "lite5" |
| installation\_way | Тип переустановки ОС на VPS. Принимает значение "auto" для автоматической переустановки ОС и значение "manual" только для заказа лицензии.  Необязательный параметр, значение по умолчанию: "manual" |
| ostmpl | Шаблон предустановленной операционной системы.  Необязательный параметр, может потребоваться в том случае если шаблон установленной ОС не поддерживается ISP Manager'ом.  Поддерживаются следующие шаблоны ОС:  centos7-x86\_64  debian9-x86\_64  ubuntu16.04-x86\_64 |

##### Дополнительный IP (srv\_addip)

| Параметр | Описание |
| --- | --- |
| uplink\_service\_id | ID родительской услуги, к которой заказывается дополнительный IP. Заказ дополнительного IP возможен только для ISPmanager хостинга (srv\_hosting\_ispmgr) и CPanel хостинга (srv\_hosting\_cpanel) |
| subtype | Тип дополнительного IP, доступны:  "ipv4" для заказа случайного доп. IP,  "other\_subnet" для заказа доп. IP находящегося в другой подсети класса C относительно основного адреса VPS-сервера.  Обязательный параметр, значение по умолчанию: "ipv4" |

##### Расширенная защита от спама (srv\_antispam)

| Параметр | Описание |
| --- | --- |
| uplink\_service\_id | ID родительской услуги, к которой заказывается расширенная защита от спама. Заказ расширенной защиты от спама возможен для домена (domain), ISPmanager хостинга (srv\_hosting\_ispmgr), CPanel хостинга (srv\_hosting\_cpanel) и Plesk хостинга (srv\_hosting\_plesk). |
| dname | Имя домена для которого заказывается расширенная защита от спама. Данный параметр может быть использован как альтернатива uplink\_service\_id. Необязательный параметр. |
| spam\_action | Данный параметр определяет какие действия будут применены к почте распознанной как спам.  Возможны 2 варианта:   * "delete" ( Удалять найденный спам ), * "mark" ( Отметить найденный спам email-заголовками X-Spam-\* ).  Необязательный параметр, значение по умолчанию: "delete". |
| mx\_list | Список MX серверов через запятую, используемых для транспорта отфильтрованной почты. |

##### Выделенный сервер (srv\_dedicated)

| Параметр | Описание |
| --- | --- |
| dedicated\_name | Наименование сервера для идентификации в списке услуг. |
| dedicated\_id | Уникальный идентификатор сервера. |
| ostmpl | Шаблон предустановленной операционной системы. Сейчас доступны: Windows Server 2019 Standard Evaluation  Windows Server 2022 Standard Evaluation  alma8  alma9  bitrix\_\_alma9  bitrix\_\_centos9-x86\_64  bitrix\_\_rocky9  centos9-x86\_64  debian11-x86\_64  debian12-x86\_64  esxi6.7  esxi7  esxi8  rocky8  rocky9  ubuntu20.04-x86\_64  ubuntu22.04-x86\_64  ubuntu24.04-x86\_64. |
| disk\_layout | Описание разметки файловой системы (в свободной форме). Необязательный параметр. |
| client\_comment | Ваши пожелания по установке сервера (в свободной форме). Необязательный параметр. |
| use\_raid | Использование raid-массива (для серверов с raid-контроллером). Необязательный параметр. |
| contype | Тип контактных данных. Принимает значение "pp" при регистрации сервера на данные физического лица и значение "org" при регистрации сервера на данные юридического лица. |
| email | e-mail адрес для создаваемого хостинг-аккаунта. |
| phone | Телефон. Необязательное поле |
| country | Двухбуквенный ISO-код страны, в которой зарегистрировано(а) физическое лицо (организация). |
| person\_r | Указывается при использовании параметра "contype" со значением "pp". Фамилия, имя и отчество администратора сервера на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации). Сокращения длиной в идин символ, в этом поле приняты не будут. Например сокращение вида Vasily N Pupkin - вызовет ошибку для данного поля.  Примеры правильного заполнения:    *Пример1: Пупкин Василий Николаевич*  *Пример2: John Smith* |
| passport | Указывается при использовании параметра "contype" со значением "pp". Серия и номер паспорта, а также наименование органа, выдавшего паспорт, и дата выдачи (в указанной последовательности, с разделением пробелами). В написании римских цифр допустимо использование только латинских букв. Дата записывается в формате ДД.ММ.ГГГГ. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются. В случае использования документа, отличного от паспорта (допустимо ТОЛЬКО для нерезидентов России), в начале строки указывается наименование вида документа. Запись может быть многострочной.  *Пример: 34 02 651241 выдан 48 о/м г.Москвы 26.12.1990* |
| org\_r | Указывается при использовании параметра "contype" со значением "hosting\_org". Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке). Запись может быть многострочной.  *Пример1: Урюпинский государственный университет\nимени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | Указывается при использовании параметра "contype" со значением "org". Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |

##### KVM доступ (srv\_kvm)

| Параметр | Описание |
| --- | --- |
| uplink\_service\_id | ID родительской услуги, к которой заказывается KVM доступ. Заказ KVM доступа возможен только для выделеного сервера (srv\_dedicated). |

##### Автоматическое SEO-продвижение (srv\_seowizard)

| Параметр | Описание |
| --- | --- |
| email | Email, для которого заказывается услуга |
| seo\_name | Имя для услуги. Необязательный параметр, если не указано, будет подставлено имя "SeoWizard для <email>" |

##### Конструктор сайтов Reg.ru (srv\_websitebuilder)

| Параметр | Описание |
| --- | --- |
| dname | Имя домена, для которого заказывается услуга конструктор Reg.ru. |
| subtype | Тарифный план, сейчас доступны: "annual", "infinite", "start-2020". |
| email | Email, для которого заказывается услуга. |

##### Пример запроса для конструктора Reg.ru:

```
https://api.reg.ru/api/regru2/service/create
dname=test.ru
email=test@test.ru
password=test
period=12
servtype=srv_websitebuilder
subtype=start
username=test
```

##### Пример ответа для конструктора Reg.ru:

```
{
   "answer" : {
      "bill_id" : "37601633",
      "descr" : "service srv_websitebuilder is ordered for domain test.ru",
      "pay_notes" : "No charge required",
      "payment" : "0.00",
      "service_id" : "27249497",
      "uplink_service_id" : "0"
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| descr | Общее описание сделанного заказа. |
| bill\_id | Номер счёта заказа. |
| payment | Цена заказа. |
| pay\_notes | Результат проведения оплаты. |
| service\_id | Числовой идентификатор услуги. |
| uplink\_service\_id | Числовой идентификатор родительской услуги. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/create
dname=qqq.ru
output_content_type=plain
password=test
period=1
plan=Host-2
servtype=srv_hosting_ispmgr
username=test
```

##### Пример ответа:

```
{
    'result' => 'success';
    'answer' => {
        'descr' => 'service srv_hosting_ispmgr is ordered for domain qqq.ru',
        'payment' => '100',
        'pay_notes' => 'Amount successfully charged',
        'bill_id' => '1234',
        'service_id' => '987654'
     }
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| NEEDS\_CONFIRMATION | Service already ordered and needs confirmation. | Услуга уже заказана и требует подтверждения. |
| WAITING | Service already ordered but not processed. | Услуга уже заказана, но еще не отработана. |
| ACCOUNT\_IS\_NOT\_TRIAL | User account $uid is not trial. Account group: $group | Аккаунт пользователя $uid не является пробным. Тип аккаунта: $group |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.5. Функция: service/check\_create

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Валидация параметров заказа новой услуги. Проверяет поле domain\_name, также может валидировать поля subtype и period.

##### [Поля запроса:](#common_input_params)

##### Основные

| Параметр | Описание |
| --- | --- |
| domain\_name | Имя домена, для которого заказывается услуга. Можно использовать сокращенный вариант параметра - dname. |

##### Прочие параметры (необязательны)

| Параметр | Описание |
| --- | --- |
| servtype | Вид заказываемой услуги. Значение по умолчанию: srv\_hosting\_ispmgr |
| subtype | Тарифный план. Значение по умолчанию: Host-0-0420 |
| period | Срок, на который заказывается услуга. Значение по умолчанию: 1 |

##### Пример запроса

```
https://api.reg.ru/api/regru2/service/check_create
dname=qqq.ru
period=1
servtype=srv_hosting_plesk
subtype=Host-1-0420
```

##### Пример ответа:

```
{
    'result' => 'success';
    'answer' => {
        'descr' => 'everything is ok for creating',
        'param' =>  {
            'dname'     => 'qqq.ru',
            'servtype'  => 'srv_hosting_plesk',
            'subtype'   => 'Host-1-0420',
            'period'    => '1'
        }
     }
}
```

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| ALREADY\_EXISTS\_ON\_SERVER | Domain already exist on hosting server | Данный домен уже присутствует на сервере |
| ALREADY\_EXISTS\_ON\_NSS | Domain already exist on hosting ns servers | Данный домен уже присутсвует на NS серверах хостинга |
| INVALID\_PERIOD | Invalid period | Неправильно задан период для услуги |
| INVALID\_PLAN | Invalid service plan | Неправильный тарифный план |
| PARAMETER\_MISSING | $param required | $param не найден(ы) |
| DOMAIN\_BAD\_NAME | Domain $domain\_name is reserved or disallowed by the registry, or is a premium domain offered by special price | Домен $domain\_name является зарезервированным или недопустимым к регистрации по правилам реестра, либо premium-доменом, предлагаемым по специальной цене |
| FIELD\_VALUE\_TOO\_SHORT | Value of field is too short | Значение поля слишком короткое |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.6. Функция: service/delete

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Удаляет услугу

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) а также

| Параметр | Значения | Описание |
| --- | --- | --- |
| servtype | "srv\_google\_apps"  "srv\_webfwd"  "srv\_parking"  "srv\_seowizard"  "srv\_websitebuilder" | Тип услуги, которую удаляем |

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/delete
domain_name=test.ru
output_content_type=plain
password=test
servtype=srv_hosting_ispmgr
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| HAS\_ACTIVE\_PROJECTS | There are active projects on your account | Имеются активные проекты |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.7. Функция: service/get\_info

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

получить информацию о услугах

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params) а также

| Параметр | Значения | Описание |
| --- | --- | --- |
| show\_folders | 0 и 1 | Дополнительно привести список папок, в которые входит услуга, по умолчанию — 0 |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | Список хешей параметров услуг. |
| subtype | подтип услуги (для хостинга: идентификатор тарифного плана) |
| state | состояние услуги, допустимые варианты:   * «N» – услуга неактивна (домен не зарегистрирован / не перенесён); * «A» – услуга активна; * «S» – услуга приостановлена; * «D» – услуга удалёна; * «O» – домен перенесён к другому регистратору. |
| creation\_date | дата активации услуги |
| expiration\_date | дата истечения оплаченного периода услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_info
input_data={"services":[{"domain_name":"test.ru"},{"service_id":"111111"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "services" : [
         {
            "creation_date" : "2001-01-01",
            "expiration_date" : "2101-01-01",
            "result" : "success",
            "service_id" : "12345",
            "servtype" : "domain",
            "state" : "A",
            "subtype" : "test"
         },
         {
            "creation_date" : "2001-01-01",
            "expiration_date" : "2101-01-01",
            "result" : "success",
            "service_id" : "111111",
            "servtype" : "domain",
            "state" : "A",
            "subtype" : "test"
         }
      ]
   },
   "result" : "success"
}
```

Если запрос выполняется от аккаунта партнера, то дополнительно отображается поле `user2_login`. Значение поля — логин аккаунта, которому услуга передана в частичное упроавление.

##### Пример ответа:

```
{
   "answer" : {
      "services" : [
         {
            "creation_date" : "2001-01-01",
            "expiration_date" : "2101-01-01",
            "result" : "success",
            "service_id" : "12345",
            "servtype" : "domain",
            "state" : "A",
            "subtype" : "test",
            "user2_login" : "test@test.com"
         },
         {
            "creation_date" : "2001-01-01",
            "expiration_date" : "2101-01-01",
            "result" : "success",
            "service_id" : "111111",
            "servtype" : "domain",
            "state" : "A",
            "subtype" : "test",
            "user2_login" : ""
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.8. Функция: service/get\_list

##### Доступность:

Клиенты

##### Назначение:

получить список активных услуг

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| servtype | Вид услуги:  domain — «Домен»,  srv\_webfwd — «Web-форвардинг»,  srv\_parking — «Парковка домена»,  srv\_dns\_both — «Поддержка DNS»,  srv\_hosting\_ispmgr — «ISPmanager хостинг»,  srv\_hosting\_cpanel — «CPanel хостинг»,  srv\_hosting\_plesk — «Plesk хостинг»,  srv\_antispam — «Расширенная защита от спама»,  srv\_addip — «Дополнительный ip адрес»,  srv\_license\_isp — «ISPmanager лицензия»,  srv\_certificate — «Сертификат на домен»,  srv\_voucher — «Свидетельство на домен».  Если значение не указано, возвращаются услуги всех видов. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | Список хешей параметров услуг. |
| service\_id | числовой идентификатор услуги |
| dname | имя домена |
| subtype | подтип услуги (для хостинга: идентификатор тарифного плана) |
| state | состояние услуги, допустимые варианты:   * «N» – услуга неактивна (домен не зарегистрирован / не перенесён); * «A» – услуга активна; * «S» – услуга приостановлена; |
| creation\_date | дата активации услуги |
| expiration\_date | дата истечения оплаченного периода услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_list
input_data={"servtype":"domain"}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "services" : [
         {
            "creation_date" : "2009-04-18",
            "dname" : "test.ru",
            "expiration_date" : "2011-04-18",
            "service_id" : "111",
            "servtype" : "domain",
            "state" : "A",
            "subtype" : "test",
            "uplink_service_id" : "0"
         },
         {
            "creation_date" : "2009-04-29",
            "dname" : "foo-test.ru",
            "expiration_date" : "2011-04-29",
            "service_id" : "222",
            "servtype" : "srv_hosting_ispmgr",
            "state" : "A",
            "subtype" : "test",
            "uplink_service_id" : "0"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.9. Функция: service/get\_folders

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получение списка папок в которые входит сервис

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| folders | список папок, может быть пустым |
| folder\_id | идентификатор папки |
| folder\_name | имя папки |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_folders
domain_name=test.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "folders" : [
         {
            "folder_id" : "12345",
            "folder_name" : "test_folder"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.10. Функция: service/get\_details

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Получения дополнительных данных по сервису, в т.ч. контактных данных для доменов

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params) а также

| Параметр | Значения | Описание |
| --- | --- | --- |
| separate\_groups | 0 и 1 | При значении 1 делается разбивка выходных данных по группам, группы для каждого сервиса или доменной зоны свои, значение по умолчанию 0 |
| show\_contacts\_only | 0 и 1 | При значении 1 возвращаются только контактные данные, что удобно при работе с доменами, для других сервисов не имеет смысла, значение по умолчанию 0 |
| splitted\_contacts | 0 и 1 | При значении 1 контактные данные .RU/.SU/.РФ доменов такие как person\_r\_surname, person\_r\_name, person\_r\_patronimic, p\_addr\_zip, p\_addr\_area, p\_addr\_city, p\_addr\_addr, p\_addr\_recipient, passport\_number, passport\_place, passport\_date, address\_r\_zip, address\_r\_area, address\_r\_city и address\_r\_addr возвращаются как есть. При значении 0 или отсуствии параметра контактные данные person\_r\_surname, person\_r\_name и person\_r\_patronimic объединяются в единый составной контакт person\_r, контактные данные p\_addr\_zip, p\_addr\_area, p\_addr\_city, p\_addr\_addr и p\_addr\_recipient объединяются в составной контакт p\_addr, контактные данные passport\_number, passport\_place и passport\_date объединяются в составной контакт passport, контактные данные address\_r\_zip, address\_r\_area, address\_r\_city и address\_r\_addr объединяются в составной контакт address\_r. Значение по умолчанию 0, но просьба использовать параметр splitted\_contacts в запросах так как со временем будут возвращаться только разделённые контактные данные. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | список доменов с параметрами dname, servtype, service\_id, details и contacts, или error\_code c кодом ошибки идентификации услуги |
| details | список всех дополнительных данных сервиса, в случае вызова функции с параметром separate\_groups делается разбивка данных по группам. |
| contacts | список контактов конкретного домена, подробное описание для каждой доменной зоны см. в описании функции [domain/create](#domain_create), возвращается только при вызове с параметром show\_contacts\_only |

##### Дополнительные поля, возвращаемые для услуги хостинга

| Поле | Описание |
| --- | --- |
| login | логин хостинг-аккаунта |
| passwd | пароль хостинг-аккаунта |
| server\_ip | IP-адрес сервера, на котором обслуживается аккаунт |
| nss | список NS-серверов хостинга |

##### Дополнительные поля, возвращаемые для услуги web-форвардинга

| Поле | Описание |
| --- | --- |
| fwds | Список всех web-перенаправлений домена. Содержит поля fwdfrom(Переадресация с), fwdto(Переадресовывать на), webfwd\_type(Способ переадресации), title(Заголовок окна, если webfwd\_type = frames). |

##### Дополнительные поля, возвращаемые для услуги парковки

| Поле | Описание |
| --- | --- |
| title | Заголовок страницы. Просто элемент оформления шаблона. |
| content | HTML-код страницы. |
| counter\_html\_code | HTML-код счётчиков (опционально). |
| template\_name | Идентификатор шаблона. Доступные идентификаторы:  private\_property  «Частная собственность»  under\_construction  «На сайте идут строительные работы»  for\_sale  «Домен выставлен на продажу»  for\_rent  «Домен сдается в аренду»  for\_you\_gift  «Для Вас подарок»  happy\_new\_year  «С Новым годом!»  happy\_birthday1  «С Днем рождения!» (вариант 1)  happy\_birthday2  «С Днем рождения!» (вариант 2)  love\_happiness  «Любви и счастья!»  empty  Пустая страница  raw\_html  Ваш HTML код (только для платной парковки, без стандарного рекламного блока) |
| html\_title | HTML мета-тег «Title». |
| html\_description | HTML мета-тег «Description». |
| html\_keywords | HTML мета-тег «Keywords». |
| opt\_user\_contacts | Отобразить информационный блок с контактами Администратора домена. |
| opt\_feedback\_link | Отобразить информационный блок со ссылкой "Связь с Администратором домена" (ссылка на Whois по домену).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_domain\_shop\_link | Отобразить информационный блок со ссылкой на лот в Магазине доменов (при условии, что домен выставлен на продажу в Магазине доменов).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_whois\_link | Отобразить информационный блок со ссылкой на историю Whois домена.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_se\_link | Отобразить информационный блок со cсылками на домен в поисковых системах.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_indexed\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_blogs\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |

##### Примеры запросов:

Запрос с несколькими доменами в списке

```
https://api.reg.ru/api/regru2/service/get_details
input_data={"username":"test","password":"test","domains":[{"dname":"vschizh.ru"},{"dname":"vschizh.org"}],"output_content_type":"plain"}
input_format=json
password=test
username=test
```

Запрос с параметром separate\_groups

```
https://api.reg.ru/api/regru2/service/get_details
domain_name=vschizh.ru
output_content_type=plain
password=test
separate_groups=1
username=test
```

Запрос с параметром show\_contacts\_only

```
https://api.reg.ru/api/regru2/service/get_details
domain_name=vschizh.ru
output_content_type=plain
password=test
show_contacts_only=1
username=test
```

##### Примеры успешного ответа:

Ответ на запрос с несколькими доменами в списке

```
{
    'answer' => {
        'services' => [
            {
                'dname'      => 'vschizh.ru',
                'servtype'   => 'domain',
                'service_id' => '12345'
                'details'    => {
                    'country'             => 'RU',
                    'e_mail'              => 'test@test.ru',
                    'person_r'            => 'Рюрик Святослав Владимирович',
                    'id_state'            => 'VERIFIED',
                    'phone'               => '+7 495 1234567',
                    'birth_date'          => '01.01.1101',
                    'descr'               => 'test contacts',
                    'person'              => 'Svyatoslav V Ryurik',
                    'p_addr'              => '12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому',
                    'passport'            => '22 44 668800, выдан по месту правления 01.09.1164'
                    'private_person_flag' => '1',
                },
                'result'     => 'success'
            },
            {
                'dname'      => 'vschizh.org',
                'servtype'   => 'domain',
                'service_id' => '12346'
                'details'    => {
                    'o_email'        => 'test@test.ru',
                    'o_addr'         => 'Vschizh Goverment, house 1, Knyazheska str',
                    'o_phone'        => '+7.4951234567',
                    'o_state'        => 'VSZ',
                    'o_postcode'     => '12345',
                    'o_city'         => 'Vschizh',
                    'o_first_name'   => 'Svyatoslav',
                    'o_last_name'    => 'Ryurik',
                    'o_company'      => 'Vschizh City',
                    'o_country_code' => 'RU',
                    'o_fax'          => '+7.4951234567'
                },
                'result'     => 'success'
            }
        ]
    },
    'result' => 'success'
}
```

Ответ на запрос с параметром separate\_groups

```
{
    'answer' => {
        'services' => [
            {
                'dname'      => 'vschizh.ru',
                'service_id' => '12345',
                'servtype'   => 'domain',
                'details'    => {
                    'ru_id' => {
                        'id_state'   => 'VERIFIED'
                    },
                    'ru_dd' => {
                        'descr'      => 'test user domain'
                    },
                    'ru_pp' => {
                        'country'    => 'RU',
                        'e_mail'     => 'test@test.ru',
                        'person_r'   => 'Рюрик Святослав Владимирович',
                        'phone'      => '+7 495 1234567',
                        'birth_date' => '01.01.1101',
                        'person'     => 'Svyatoslav V Ryurik',
                        'p_addr'     => '12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому',
                        'passport'   => '22 44 668800, выдан по месту правления 01.09.1164'
                    }
                },
                'result'     => 'success'
            }
        ]
    },
    'result' => 'success'
}
```

Ответ на запрос с параметром show\_contacts\_only

```
{
    'answer' => {
        'services' => [
            {
                'contacts' => {
                    'country'    => 'RU',
                    'e_mail'     => 'test@test.ru',
                    'person_r'   => 'Рюрик Святослав Владимирович',
                    'phone'      => '+7 495 1234567',
                    'birth_date' => '01.01.1101',
                    'descr'      => 'test user domain'
                    'person'     => 'Svyatoslav V Ryurik',
                    'p_addr'     => '12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому',
                    'passport'   => '22 44 668800, выдан по месту правления 01.09.1164'
                },
                'dname'      => 'vschizh.ru',
                'service_id' => '12345',
                'servtype'   => 'domain',
                'result'     => 'success'
            }
        ]
    },
    'result' => 'success'
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.11. Функция: service/get\_dedicated\_server\_list

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение списка выделенных серверов доступных для заказа.

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| server\_id | Уникальный идентификатор сервера |
| cpu\_content | Описание модели процессора |
| cpu\_count | Количество процессоров |
| cpu\_core | Количество ядер на процессор |
| ram\_content | Тип оперативной памяти |
| ram\_count | Количество плат памяти |
| ram\_size | Объем памяти одной платы |
| hdd\_content | Тип интерфейса жесткого диска |
| hdd\_count | Количество жестких дисков |
| hdd\_size | Объем одного жесткого диска |
| month\_traf | Количество предоплаченного трафика, гб в месяц |
| price\_retail | Стоимость аренды сервера в месяц |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_dedicated_server_list
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
  answer => {
      server_list => [
          {
              server_id => "171",
              cpu_content => "Intel Pentium 4 2.8Ghz",
              cpu_count => "1",
              cpu_core => "2",
              hdd_content => "IDE",
              hdd_count => "2",
              hdd_size => "120Gb",
              ram_content => "RAM"
              ram_count => "2",
              ram_size => "512Mb",
              month_traf => "1000",
              price_retail => "3080",
          }
      ]
  },
  result => "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.12. Функция: service/update

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

настройка услуги.

##### [Поля запроса:](#common_input_params)

##### Общие

| Параметр | Описание |
| --- | --- |
| dname | Имя домена настраиваемого сервиса. |
| servtype | Вид заказываемой услуги:  srv\_webfwd —«Web-форвардинг»,  srv\_parking —«Парковка»,  srv\_ssl\_certificate—«SSL-Сертификат»,  srv\_hosting\_cpanel —«Cpanel хостинг»,  srv\_antispam —«Расширенная защита от спама»,  srv\_dedicated —«Выделенный сервер». |

##### Web-форвардинг (srv\_webfwd)

| Параметр | Описание |
| --- | --- |
| fwd\_action | Действия, доступны:   * addfwd ( добавить перенаправление ) * rmfwd ( удалить перенаправление ) * rmall ( удалить все перенаправления ) * edit\_allow\_se\_indexing ( индексация поисковыми системами ) * add\_favicon ( добавить favicon для переадресаций с маскировкой адреса во фрейме ) * rm\_favicon ( удалить favicon для переадресаций с маскировкой адреса во фрейме )  DEPRECATED. Используйте параметр subtask. |
| subtask | Действие, доступны:   * addfwd ( добавить перенаправление ) * rmfwd ( удалить перенаправление ) * rmall ( удалить все перенаправления ) * edit\_allow\_se\_indexing ( индексация поисковыми системами ) * add\_favicon ( добавить favicon для переадресаций с маскировкой адреса во фрейме ) * rm\_favicon ( удалить favicon для переадресаций с маскировкой адреса во фрейме ) |
| fwdfrom | «Переадресация с», укажите относительный адрес (без имени домена), с которого требуется осуществлять перенаправление(если не указано, то «/»). |
| fwdto | «Переадресовывать на», укажите URL, на который следует перенаправлять посетителей. |
| allow\_se\_indexing | Параметр указывает должна ли страница индексироваться поисковыми системами. Имеет смысл только в случае наличия как минимум одного перенаправления. Возможные значения:   * «1» — функция включена * «0» — функция отключена  Если не указано, то «0». |
| webfwd\_type | Способ переадресации, может принимать значения:   * redirect ( перенаправление запроса с ипользованием HTTP заголовка с кодом 301 ) * frames ( маскировка адреса во фрейме )  Если не указано, то «redirect». |
| title | Заголовок окна, имеет смысл только в случае использования маскировки адреса во фрейме. Указанный заголовок будет заголовком страницы (будет отображаться в качестве заголовка окна браузера). |
| favicon | Favicon для переадресаций, имеет смысл только в случае использования маскировки адреса во фрейме. Допускается только картинка в формате PNG закодированная в Base64. Исходный размер картинки 16x16 пикселей и не более 1536 байт. Используется совместно с параметром subtask = add\_favicon |

##### Парковка (srv\_parking)

| Параметр | Описание |
| --- | --- |
| title | Заголовок страницы. Просто элемент оформления шаблона. |
| content | HTML-код страницы. |
| counter\_html\_code | HTML-код счётчиков (опционально). |
| template\_name | Идентификатор шаблона. Доступные идентификаторы:  private\_property  «Частная собственность»  under\_construction  «На сайте идут строительные работы»  for\_sale  «Домен выставлен на продажу»  for\_rent  «Домен сдается в аренду»  for\_you\_gift  «Для Вас подарок»  happy\_new\_year  «С Новым годом!»  happy\_birthday1  «С Днем рождения!» (вариант 1)  happy\_birthday2  «С Днем рождения!» (вариант 2)  love\_happiness  «Любви и счастья!»  empty  Пустая страница  raw\_html  Ваш HTML код (только для платной парковки, без стандарного рекламного блока) |
| html\_title | HTML мета-тег «Title». |
| html\_description | HTML мета-тег «Description». |
| html\_keywords | HTML мета-тег «Keywords». |
| opt\_user\_contacts | Отобразить информационный блок с контактами Администратора домена. |
| opt\_feedback\_link | Отобразить информационный блок со ссылкой "Связь с Администратором домена" (ссылка на Whois по домену).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_domain\_shop\_link | Отобразить информационный блок со ссылкой на лот в Магазине доменов (при условии, что домен выставлен на продажу в Магазине доменов).  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_whois\_link | Отобразить информационный блок со ссылкой на историю Whois домена.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_se\_link | Отобразить информационный блок со cсылками на домен в поисковых системах.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_indexed\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| opt\_blogs\_link | Отобразить информационный блок cо ссылками на проиндексированные страницы различными поисковыми системами.  Возможные значения:   * «1» — функция включена * «0» — функция отключена |
| subtype | Необязательный. Для бесплатной парковки следует указать значение free |

##### Сертификат SSL (srv\_ssl\_certificate)

| Параметр | Описание |
| --- | --- |
| **обязательные:**  org\_org\_name, org\_address, org\_city, org\_state, org\_postal\_code, org\_country, org\_phone    **необязательные:**  org\_first\_name, org\_last\_name, org\_fax | Название, имя, фамилия, адрес, город, штат(провинция), почтовый индекс, страна, телефон, факс организации.    Используются для всех SSL сертификатов компаний Thawte, Comodo, VeriSign, а так-же для GeoTrust SSL сертификатов: True BusinessID, True BusinessID Wildcard, True BusinessID with EV. |
| **обязательные:**  admin\_first\_name, admin\_last\_name, admin\_title, admin\_address, admin\_city, admin\_state, admin\_postal\_code, admin\_country, admin\_phone, admin\_email,    **необязательные:**  admin\_fax, admin\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес администратора.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust, Trustwave и VeriSign, а так-же для Comodo EV SSL сертификата. |
| **обязательные:**  billing\_first\_name, billing\_last\_name, billing\_title, billing\_address, billing\_city, billing\_state, billing\_postal\_code, billing\_country, billing\_phone, billing\_email,    **необязательные:**  billing\_fax, billing\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес финансового менеджера.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust и VeriSign. |
| **обязательные:**  tech\_first\_name, tech\_last\_name, tech\_title, tech\_address, tech\_city, tech\_state, tech\_postal\_code, tech\_country, tech\_phone, tech\_email,    **необязательные:**  tech\_fax, tech\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес технического специалиста.    Используются для всех SSL сертификатов компаний Thawte, GeoTrust и VeriSign. |
| **обязательные:**  signer\_first\_name, signer\_last\_name, signer\_title, signer\_address, signer\_city, signer\_state, signer\_postal\_code, signer\_country, signer\_phone, signer\_email,    **необязательные:**  signer\_fax, signer\_org\_name | Имя, фамилия, должность, адрес, город, штат, почтовый индекс, страна, телефон, e-mail, факс, название организации адрес ответственного за подписку SSL сертиификата.    Используются для Comodo EV SSL сертификата. |
| approver\_email | E-mail адрес для подтверждения сертификата. |
| server\_type | Программное обеспечение. Возможные значения:  Для Comodo SSL сертификатов:  apachessl, citrix, domino, ensim, hsphere, iis4, iis6, iis7, iplanet, javawebserver, netscape, ibmhttp, novell, oracle, other, plesk, redhat, sap, tomcat, webstar, whmcpanel    Для Thawte, GeoTrust и VeriSign SSL сертификатов:  apache2 apacheopenssl apacheraven apachessl apachessleay c2net cobaltseries cobaltraq3 cobaltraq2 cpanel domino dominogo4626 dominogo4625 ensim hsphere iis iis4 iis5 iplanet ipswitch netscape ibmhttp other plesk tomcat weblogic website webstar webstar4 zeusv3    Для TrustWave SSL Сертификатов данный параметр не используется. |
| csr\_string | Закодированный CSR включая маркеры начала и окончания. |
| subtype | Вид сертификата. Возможные значения:    **Thawte:**  ssl123, sgcsuper\_certs, sslwebserver, sslwebserver\_wildcard, sslwebserver\_ev    **Symantec:**  securesite, securesite\_pro, securesite\_ev, securesite\_pro\_ev    **GeoTrust:**  quickssl, quickssl\_premium, truebizid, truebizid\_wildcard, truebizid\_ev    **TrustWave:**  trustwave\_dv, trustwave\_ev, trustwave\_premiumssl, trustwave\_premiumssl\_wildcard    **Comodo:**  comodo\_ev, comodo\_instantssl, comodo\_premiumssl, comodo\_premiumssl\_wildcard, comodo\_ssl, comodo\_wildcard |
| reissue | Ненулевое значение инициирует переиздание сертификата с новыми полями csr\_string, server\_type, approver\_email.  Нулевое значение или отсутствие флага вызывает обновление полей, без переиздания сертификата.  Такой режим используется для подготовки полей перед продлением сертификата. |

##### Cpanel хостинг (srv\_hosting\_cpanel)

| Параметр | Описание |
| --- | --- |
| subtask | Действие, доступны:  change\_password(сменить пароль на хостинг). |

##### Расширенная защита от спама (srv\_antispam)

| Параметр | Описание |
| --- | --- |
| subtask | Действие, доступны:   * change\_action(сменить параметр "action", определяющий действия выполняемые над почтой распознанной как спам), * change\_mx\_list(сменить список mx серверов используемых для транспорта отфильтрованной почты). |
| spam\_action | Действие над спамом. Определяет какие действия будут выполнены над почтой распознанной как спам.  Возможны 2 варианта:   * "delete" ( Удалять найденный спам ), * "mark" ( Отметить найденный спам email-заголовками X-Spam-\* ).  Необязательный параметр, значение по умолчанию: "delete".  Актуально для subtask = change\_action. |
| mx\_list | Список mx серверов через запятую, используемых для транспорта отфильтрованной почты.  Актуально для subtask = change\_mx\_list. |

##### Выделенный сервер (srv\_dedicated)

| Параметр | Описание |
| --- | --- |
| subtask | Действие, доступны:   * change\_hostname(сменить обратную PTR запись для одного из ip-адресов сервера ). |
| revert\_dns\_ip | Один из ip-адресов сервера. |
| new\_hostname | Новое доменное имя для указанного ip. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| descr | Общее описание сделанного заказа. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/update
dname=qqq.ru
fwd_action=addfwd
fwdfrom=/
fwdto=http://www.reg.ru
output_content_type=plain
password=test
servtype=srv_webfwd
username=test
webfwd_type=redirect
```

##### Пример ответа:

```
{
   "answer" : {
      "descr" : "service srv_webfwd is updated for domain qqq.ru"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.13. Функция: service/renew

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

продление домена или услуги

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| period | Период продления. |
| pay\_type  ok\_if\_no\_money | См. [Общие параметры оплаты](#common_payment_params). |
| allow\_create\_bills | Флаг, указывающий, что в случае недостатка денег на счёте, запрос будет завершен без ошибки — будет создана выписка пополнения через банк.  DEPRECATED. Не использовать! |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| period | Установленный период продления. |
| bill\_id | Номер счёта заказа. |
| payment | Цена заказа. |
| currency | Валюта указанной цены. |
| status | Состояние заказа. Допустимые варианты: renew\_success и only\_bill\_created для случаев успешного прохождения счёта и нехватки денег соответственно. |

##### Примеры запросов:

Идентификация по service\_id

```
https://api.reg.ru/api/regru2/service/renew
output_content_type=plain
password=test
period=2
service_id=123456
username=test
```

Идентификация по servtype + domain\_name

```
https://api.reg.ru/api/regru2/service/renew
domain_name=regrupanel.ru
output_content_type=plain
password=test
period=1
servtype=srv_hosting_ispmgr
username=test
```

##### Примеры успешных ответов:

Ответ на "PLAIN"-запрос

```
{
    'answer' => {
        'period'   => '2',
        'payment'  => '100',
        'currency' => 'RUR',
        'status'   => 'renew_success',
        'dname'    => 'test12345.ru',
        'bill_id'  => '123456',
        'servtype' => 'domain'
    },
    'result' => 'success',
}
```

Ответ на список из 2х доменов в JSON формате

```
{
    'answer' => {
        'currency' => 'RUR',
        'payment'  => '200',
        'period'   => '2',
        'services' => [
            {
                'dname'      => 'test12345.ru',
                'service_id' => '12345',
                'servtype'   => 'domain',
                'result'     => 'success'
            },
            {
                'dname'      => 'test12346.ru',
                'service_id' => '12346',
                'servtype'   => 'domain',
                'result'     => 'success'
            }
        ],
        'bill_id'  => '123456',
        'status'   => 'renew_success',
    },
    'result' => 'success',
}
```

Ответ при передаче флага `allow_create_bills` (случай, когда не хватило денег и создан лишь счёт):

```
{
    'answer' => {
        'period'   => '1',
        'payment'  => '100',
        'currency' => 'RUR',
        'status'   => 'only_bill_created',
        'dname'    => 'test12345.ru',
        'bill_id'  => '123123',
        'servtype' => 'domain'
    },
    'result' => 'success',
}
```

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INCORRECT\_STATE | Operation allowed only for suspended or active services | Операция воможна только для активной или приостановленной услуги |
| PROLONG\_ERROR | Prolong error: $error\_detail | Ошибка продления: $error\_detail |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.14. Функция: service/get\_bills

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

получить список счетов, связанных с указанными услугами

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации доменов и услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | список запрошенных сервисов |
| bills | список номеров счетов, относящихся к данному сервису |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_bills
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "services" : [
         {
            "bills" : [
               "123456",
               "234567"
            ],
            "dname" : "qqq.ru",
            "service_id" : "12345",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.15. Функция: service/check\_readonly

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

проверка разрешено ли редактировать сервис (домен)

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации доменов и услуг](#common_service_identification_params)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

проверить разрешение на редактирование сервисам с ID 999999, 123456, и 654321

```
https://api.reg.ru/api/regru2/service/check_readonly
output_content_type=plain
password=test
service_id=654321
username=test
```

##### Пример ответа:

```
"answer" => {
    "services" => [
        {
            "error_code" : "SERVICE_ID_NOT_FOUND",
            "error_detail" : "SERVICE_ID_NOT_FOUND",
            "is_readonly" : 1,
            "service_id" : "999999"
        },
        {
            "error_code" : "DATA_IS_READ_ONLY",
            "error_detail" : "DATA_IS_READ_ONLY",
            "is_readonly" : 1,
            "service_id" : "654321"
        },
        {
            "error_code" : null,
            "error_detail" : null,
            "is_readonly" : 0,
            "service_id" : "123456"
        }
    ],
    "result" => "success"
}
```

В данном примере, сервис с ID = 999999 не существует, сервис с ID = 654321 недоступен к редактированию например, из за отсутствия необходимых документов, а сервис с ID = 123456 не имеет блокировок и доступен к редактированию.

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.16. Функция: service/set\_autorenew\_flag

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

установить или снять флаг автопродления.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| flag\_value | Флаг для установки, допустимые значения 0 и 1, любое ненулевое значение считается равным 1 |

А так же [стандартные параметры идентификации доменов и услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

Стандартные поля ответов для успешного изменения и ошибок

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/set_autorenew_flag
dname=qqq.ru
flag_value=1
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.17. Функция: service/suspend

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

приостановить действие услуги (для домена - снять с делегирования)

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/suspend
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.18. Функция: service/resume

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

возобновить действие услуги (для домена - делегировать)

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/resume
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| SERVICE\_NOT\_SUSPENDED | Service not suspended | Услуга не приостановлена |
| SERVICE\_EXPIRED | Service expired | Срок действия услуги истек |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.19. Функция: service/get\_depreciated\_period

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

расчитать число периодов до даты истечения срока действия услуги

##### [Поля запроса:](#common_input_params)

[Cтандартные параметры идентификации доменов и услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| depreciated\_period | Дробное число периодов, оставшихся до даты истечения срока действия услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/get_depreciated_period
dname=qqq.ru
output_content_type=plain
password=test
servtype=srv_hosting_ispmgr
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "depreciated_period" : "1"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.20. Функция: service/upgrade

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

произвести повышение подтипа (тарифа) услуги. Используется для изменения тарифа виртуального хостинга ("srv\_hosting\_ispmgr") и конструктора сайтов REG.RU ("srv\_websitebuilder").

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subtype | Новый подтип (тариф) услуги. Для услуг хостинга ("srv\_hosting\_ispmgr") допустимыми значениями являются: "Host-0-0420", "Host-1-0420", "Host-3-0420", "Host-A-0420", "Host-B-0420", "Host-Lite-0420".  Для услуги конструктор сайтов REG.RU ("srv\_websitebuilder") допустимыми значениями являются: "start", "standart", "premium". |
| servtype | Вид услуги. srv\_hosting\_cpanel  srv\_hosting\_ispmgr  srv\_hosting\_plesk  srv\_google\_apps |
| period | Число периодов, на который заказывается новая услуга |

А так же [стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| withdrawed\_amount | Количество денежных средств, списанных по операции |
| returned\_amount | В том числе зачислено на лицевой счет |
| new\_service\_id | Идентификатор нового сервиса с новым подтипом (тарифом) |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/upgrade
dname=qqq.ru
output_content_type=plain
password=test
period=2
servtype=srv_hosting_ispmgr
subtype=Host-2
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "bill_id" : "123",
      "new_service_id" : "-1",
      "returned_amount" : "100",
      "withdrawed_amount" : "100"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| SERVICE\_UPGRADE\_NOT\_ALLOWED | Service upgrade is not allowed | Повышение тарифа для данного сервиса запрещено |
| DISK\_SIZE\_TOO\_LARGE | Additional disk space is too large | Указан слишком большой размер дополнительного места |
| SERVICE\_BLOCKED | Service blocked | Сервис заблокирован |

Cм. [cтандартные коды ошибок](#common_errors)

## 7.21. Функция: service/partcontrol\_grant

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

предоставить право частичного управления услугой другому пользователю

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| newlogin | Логин пользователя, которому нужно передать частичное управление |

А также [стандартные параметры идентификации доменов и услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| newlogin | Логин пользователя, которому передано частичное управление |
| service\_id | идентификатор домена или услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/partcontrol_grant
newlogin=test_user
output_content_type=plain
password=test
service_id=1
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "newlogin" : "test_user",
      "service_id" : "1"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.22. Функция: service/partcontrol\_revoke

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

отключить право частичного управления услугой

##### [Поля запроса:](#common_input_params)

См. [стандартные параметры идентификации доменов и услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| service\_id | идентификатор домена или услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/partcontrol_revoke
output_content_type=plain
password=test
service_id=1
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "service_id" : "1"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.23. Функция: service/resend\_mail

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

повторить email сообщение.

##### [Поля запроса:](#common_input_params)

[Стандартные параметры идентификации услуг](#common_service_identification_params). Помимо этого для услуги SSL-сертификата необходимо указывать тип пересылаемого сообщения mailtype.

| Параметр | Описание |
| --- | --- |
| mailtype | Тип email сообщения:  approver\_email — сообщение для потверждения заказа сертификата,  certificate\_email — сообщение с сертификатом. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| service\_id | идентификатор домена или услуги |
| dname | имя домена сервиса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/service/resend_mail
output_content_type=plain
password=test
servtype=srv_ssl_certificate
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "dname" : "test.ru",
      "result" : "success",
      "service_id" : "111"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Пример ошибки для не поддерживаемых услуг:

```
{
    error_text : 'Mail resend allowed only for hosting and ssl certificates',
    error_code : 'ONLY_HOSTING_AND_SSL_CERTIFICATES',
    result : 'error'
}
```

Пример ошибки в случае если не возможно повторно отправить `email`:

```
{
    error_code : "CAN_NOT_RESEND_EMAIL"
    result : 'error'
}
```

Cм. [cтандартные коды ошибок](#common_errors)

## 7.24. Функция: service/refill

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Пополнение баланса услуги. Поддерживаемые типы услуг:

* srv\_seowizard — Автоматическое SEO-продвижение (минимальная сумма пополнения - 10 рублей.)

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bill\_id | Номер счёта заказа. |
| pay\_type | Способ оплаты. |
| currency | Валюта указанной цены. |
| payment | Цена заказа. |
| pay\_notes | Комментарий, относящийся к используемому способу оплаты. |
| service\_id | Идентификатор услуги. |

##### Пример запроса:

идентификация по service\_id

```
https://api.reg.ru/api/regru2/service/refill
amount=10
currency=UAH
output_content_type=plain
password=test
service_id=123456
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "bill_id" : "1000000",
      "currency" : "RUR",
      "pay_notes" : "Amount successfully charged",
      "pay_type" : "prepay",
      "payment" : "36.88",
      "service_id" : "123456"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.25. Функция: service/get\_balance

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение баланса внешней услуги. Поддерживаемые типы услуг: srv\_seowizard — Автоматическое SEO-продвижение

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример ответа:

```
{
   "answer" : {
      "balance" : "10",
      "currency" : "USD"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.26. Функция: service/seowizard\_manage\_link

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение ссылки на панель управления услугой "Автоматическое SEO-продвижение"

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример ответа:

```
{
   "answer" : {
      "link" : "http://reg.ru/seowizard.html"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 7.27. Функция: service/get\_websitebuilder\_link

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение ссылки для публикации конструктора

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуг](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример ответа:

```
{
   "answer" : {
      "link" : "http://www.reg.ru/ru/?login_hash=123"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 8. Функции для работы с доменами

## 8.1. Функция: domain/nop

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

для тестирования, также позволяет проверить доступность домена и получить его id, если передать username+password+dname

##### [Поля запроса:](#common_input_params)

отсутствуют или [стандартные параметры идентификации доменов](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| service\_id | идентификатор домена, присутствует только при передаче имени домена в поле domain\_name/dname |

##### Пример запроса:

проверка доступности API

```
https://api.reg.ru/api/regru2/domain/nop
output_content_type=plain
```

проверка существования домена с получением его id

```
https://api.reg.ru/api/regru2/domain/nop
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.2. Функция: domain/get\_prices

##### Доступность:

Все, кроме индивидуальных тарифных планов [Рег. решения](https://www.reg.ru/vybor-tarifnogo-plana/)

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение цен на регистрацию/продление доменов во всех доступных зонах.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| [Параметры для аутентификации](#common_auth_params) | используйте аутентификацию для получения партнёрских цен. |
| show\_renew\_data | флаг возврата цен для продления регистрации (1/0). Необязательное поле. |
| show\_update\_data | Флаг возврата цен для обновления домена (1/0). Необязательное поле. |
| currency | Идентификатор валюты, в которой будут возвращаться цены (RUR, UAH, USD, EUR). Необязательное поле, по умолчанию цены указываются в рублях. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| currency | валюта, в которой возвращены цены |
| price\_group | тарифный план |
| last\_update\_time | дата-время последнего обновления цен для данного тарифного плана |
| prices | цены доменнов по зонам  Цены указаны за год. Для некоторых зон минимальный срок регистрации больше года.  Если для зоны разрешена регистрация русских имен доменов, и она отличается от цены для регистрации доменов с использованием только латинских букв, то для этой зоны возвращается дополнительная запись с префиксом '\_\_idn.'. |
| renew\_price | стоимость продления для вашего тарифа |
| retail\_renew\_price | стоимость продления для пользователя со статусом "Розница" |
| reg\_price | стоимость регистрации для вашего тарифа |
| retail\_reg\_price | стоимость регистрации для пользователя со статусом "Розница" |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_prices
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "currency" : "RUR",
      "price_group" : "Retail",
      "prices" : {
         "__idn.com" : {
            "reg_max_period" : "10",
            "reg_min_period" : "1",
            "reg_price" : "960"
         },
         "com" : {
            "reg_max_period" : "10",
            "reg_min_period" : "1",
            "reg_price" : "450"
         },
         "ru" : {
            "reg_max_period" : "1",
            "reg_min_period" : "1",
            "reg_price" : "590"
         },
         "рф" : {
            "reg_max_period" : "1",
            "reg_min_period" : "1",
            "reg_price" : "1200"
         }
      }
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.3. Функция: domain/get\_suggest

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

подбор имени для домена по ключевым словам, функция работает подобно сервису Reg.Choice

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| word | Главное ключевое слово, например «дом» или «domain». Обязательное поле. |
| additional\_word | Дополнительное ключевое слово, например «новый» или «cool». Необязательное поле. |
| category | Категория подбираемых имён. Может принимать значения: 'pattern' — шаблонные («имя + префикс»), 'search\_trends' — поисковые тренды, 'all' — все. По умолчанию возвращаются имена из всех категорий. |
| tlds | Зона, в которой проверяется доступность доменного имени к регистрации. Зоны могут быть такими: 'com', 'info', 'name', 'net', 'org', 'ru', 'ru.com', 'su', 'рф'. Для задания нескольких зон одновременно, необходимо добавить в запрос это поле для каждой зоны: например "...&tlds=ru&tlds=su&tlds=com". Если поле не задано ни разу, то доступность проверяется во всех перечисленных выше зонах. |
| use\_hyphen | Если значение истинно, использовать дефис в для разделения отдельных слов в доменном имени ("cool-domain"). По умолчанию слова склеиваются без разделителя ("cooldomain"). |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| suggestions | Массив вариантов имён для доменов. Каждый элемент в массиве содержит поле "name" (вариант имени) и массив "avail\_in", в котором перечислены зоны, где такое доменное имя доступно к регистрации. Максимальный размер массива — 100. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_suggest
additional_word=новый
output_content_type=plain
password=test
tlds=рф
use_hyphen=1
username=test
word=дом
```

##### Пример ответа:

```
{
   "answer" : {
      "suggestions" : [
         {
            "avail_in" : [
               "рф",
               "su"
            ],
            "name" : "дом"
         },
         {
            "avail_in" : [
               "ru",
               "su"
            ],
            "name" : "dom"
         },
         {
            "avail_in" : [
               "рф",
               "su"
            ],
            "name" : "новый"
         },
         {
            "avail_in" : [
               "ru",
               "su",
               "org"
            ],
            "name" : "novii"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.4. Функция: domain/get\_premium\_prices

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

возвращает цены для указанных премиум-доменов. Функция проверяет только домены, которые регистратура определила как премиальные и разрешает регистрировать только по повышенной стоимости.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domains | Список доменов (не более 100 штук) |
| currency | Валюта |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Массив хешей с премиум-доменами. Каждый хеш в массиве содержит поле \"domain\_name\" (домен), поле \"register\_price\" (цена регистрации), renew\_price, register\_period, renew\_period |
| currency | Валюта |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_premium_prices
input_data={"currency":"USD","domains":["ua.ae.org", "007.academy"]}
input_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "currency" : "USD",
      "domains" : [
         {
            "domain_name" : "007.academy",
            "register_period" : "1",
            "register_price" : "238.16",
            "renew_period" : "1",
            "renew_price" : "238.16"
         },
         {
            "domain_name" : "ua.ae.org",
            "register_period" : "1",
            "register_price" : "280.51",
            "renew_period" : "0",
            "renew_price" : "0.000"
         }
      ]
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.5. Функция: domain/get\_deleted

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Получение списка освобождённых доменов, функция работает подобно странице [Свободные и удаленные домены](/domain/deleted/)

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| tlds | Зоны, в которых искать освобождённые домены. Зоны могут быть такими: 'ru', 'рф', 'su'. Для задания нескольких зон одновременно, необходимо добавить в запрос это поле для каждой зоны: например "...&tlds=ru&tlds=рф". Если поле не задано ни разу, то поиск идёт во всех перечисленных выше зонах. |
| deleted\_from | Дата удаления доменов, начиная с которой включать домены в результирующий список. Формат даты: 'ГГГГ-ММ-ДД'. По умолчанию равна последней дате удаления доменов. |
| deleted\_to | Дата удаления доменов, заканчивая которой включать домены в результирующий список. Формат даты: 'ГГГГ-ММ-ДД'. По умолчанию равна последней дате удаления доменов. Значение 'deleted\_to' не может быть позже значения 'deleted\_from' более чем на два месяца. |
| created\_from | Дата первичной регистрации доменов, начиная с которой включать домены в результирующий список. Формат даты: 'ГГГГ-ММ-ДД'. |
| created\_to | Дата первичной регистрации доменов, заканчивая которой включать домены в результирующий список. Формат даты: 'ГГГГ-ММ-ДД'. |
| hidereg | При значении 1 возвращаются только свободные на данный день домены из общего списка освобождённых, по умолчанию — 0. |
|  |  |
| min\_pr | Минимальное значение Google PR домена |
| min\_cy | Минимальное значение Яндекс тИЦ домена |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Список доменов с результатом. Содержит поля: domain\_name — имя домена, date\_delete — дата удаления, registered — статус текущей регистрации домена: 'NOT REGISTERED' (свободен) или 'REGISTERED' (занят), first\_create\_date — дата первичной регистрации домена, yandex\_tic — значение Яндекс тИЦ домена, google\_pr — значение Google PR домена. Максимальный размер списка — 50000. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_deleted
min_pr=1
password=test
tlds=ru
username=test
```

##### Пример ответа:

```
{
   "answer" => {
      "domains" => [
         {
            "domain_name" => "apmatypa.ru",
            "registered" => "NOT REGISTERED",
            "yandex_tic" => 0,
            "first_create_date" => "2006-05-18",
            "google_pr" => "2",
            "date_delete" => "2013-09-26"
         },
         {
            "domain_name" => "hudojnick.ru",
            "registered" => "NOT REGISTERED",
            "yandex_tic" => 0,
            "first_create_date" => "2004-10-19",
            "google_pr" => "1",
            "date_delete" => "2013-09-26"
         },
         {
            "domain_name" => "dpr-ryazan.ru",
            "registered" => "NOT REGISTERED",
            "yandex_tic" => 0,
            "first_create_date" => "2007-07-23",
            "google_pr" => "1",
            "date_delete" => "2013-09-26"
         }
      ]
   }
   "result" => "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.6. Функция: domain/check

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

проверка доступности регистрации домена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domain\_name | Имя домена, поле не совместимо со спиcком domains. |
| domains | Массив со списком вариантов имён доменов, каждый элемент массива является хешем с ключём dname или domain\_name. Используется только в случае запроса в форматах JSON, XML. |
| is\_transfer | При значении 1 делается проверка на возможность переноса домена в REG.RU, при нулевом значении — обычная проверка на возможность регистрации, по умолчанию — 0 |
| premium\_as\_taken | По всем премиум-доменам отвечать что домен занят (0/1). Необязательное поле |
| currency | Валюта в которой будут возвращаться цены (RUR, UAH, USD, EUR). Необязательное поле, по умолчанию цены указываются в рублях |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Массив со списком хешей, содержащий имена доменов dname и их доступность. При положительном ответе поле result будет иметь значение Available. |
| is\_premium | Флаг показывающий что это премиум-домен, при указании premium\_as\_taken=1 возвращается всем премиумам |
| price | Цена регистрации домена на 1 год, отображается только доступного к регистрации у премиум-домена с флагом is\_premium |
| renew\_price | Цена продления домена на 1 год, отображается только доступного к регистрации у премиум-домена с флагом is\_premium |
| currency | Валюта в которой возвращаться цены |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/check
input_data={"domains":[{"dname":"ya.ru"},{"dname":"yayayayayaya.ru"},{"dname":"xn--000.com"},{"dname":"china.cn"},{"dname":"ййй.me"},{"dname":"wwww.ww"},{"dname":"a.ru"},{"dname":"qqйй.com"},{"dname":"rr.ru.com"}]}
input_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "ya.ru",
            "error_code" : "DOMAIN_ALREADY_EXISTS",
            "result" : "Domain already exists, use whois service"
         },
         {
            "dname" : "yayayayayaya.ru",
            "result" : "Available"
         },
         {
            "dname" : "xn--000.com",
            "error_code" : "INVALID_DOMAIN_NAME_PUNYCODE",
            "result" : "Invalid punycode value for domain_name"
         },
         {
            "dname" : "china.cn",
            "error_code" : "TLD_DISABLED",
            "result" : "Registration in .cn TLD is not available"
         },
         {
            "dname" : "ййй.me",
            "error_code" : "DOMAIN_BAD_NAME",
            "result" : "Invalid domain name=> ййй.me"
         },
         {
            "dname" : "wwww.ww",
            "error_code" : "INVALID_DOMAIN_NAME_FORMAT",
            "result" : "domain_name is invalid or unsupported zone"
         },
         {
            "dname" : "a.ru",
            "error_code" : "DOMAIN_INVALID_LENGTH",
            "result" : "Invalid domain name length, You have entered too short or too long name"
         },
         {
            "dname" : "qqйй.com",
            "error_code" : "HAVE_MIXED_CODETABLES",
            "result" : "You can not mix latin and cyrillic letters in domain names"
         },
         {
            "currency" : "RUR",
            "dname" : "rr.ru.com",
            "is_premium" : "1",
            "price" : "60000",
            "renew_price" : "600",
            "result" : "Available"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.7. Функция: domain/create

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

подать заявку на регистрацию домена.

##### [Поля запроса](#common_input_params)

| Поле | Мин. длина | Макс. длина | Описание |
| --- | --- | --- | --- |
| domain\_name | зависят  от зоны | | Имя регистрируемого домена. Допустимые символы (есть или нет IDN) и длина поля зависят от доменной зоны. Обязательное поле.  P.S. Возможность оптовой регистрации (большое количество доменов одной зоны) одним запросом для VIP-клиентов. |
| domains | — | | Массив со списком доменов; каждый элемент массива является хешем с ключами dname — имя домена и дополнительными сервисами для каждого домена (srv\_certificate, srv\_parking, srv\_webfwd), если они нужны. Несовместимо с передачей имени домена через domain\_name. |
| period | 1 | 2 | Период, на который производится регистрация домена, допустимые значения зависят от доменной зоны, например, для .ru и .su только 1. Обязательное поле. |
| enduser\_ip | 11 | 15 | IP-адрес конечного пользователя (пользователя, который сделал заказ). Обязательное поле для партнёра, у обычного клиента по умолчанию подставляется IP-адрес с которого делается запрос. |
| contacts | — | | Группирующий хеш полей контактных данных. Список полей зависит от регистрируемой зоны и/или является ли будущий владелец домена физ. или юридическим лицом. Применим только при передаче данных в JSON или XML форматах. |
| profile\_type | — | | Тип профиля контактных данных пользователя. Параметр не совместим с явным указанием контактных данных (имеет больший приоритет — перезаписывает их) и требует указания имени профиля profile\_name. На данный момент допустимы такие варианты: GTLD, EU, TJ, RU.PP, RU.ORG, NEWTCI.ORG, NEWTCI.PP. В дальнейшем их количество будет увеличено. |
| profile\_name | — | | Имя профиля контактных данных пользователя. Параметр не совместим с явным указанием контактных данных (имеет больший приоритет — перезаписывает их) и требует указания имени профиля profile\_type. Для разных типов профилей имена могут повторяться. На данные момент создаются профили только через web-интерфейс. |
| nss | — | | Группирующий хеш полей имён и IP-адресов NS-серверов. Применим только при передаче данных в JSON или XML форматах. |
| not\_delegated | 1 | | При выставлении этого флага для .ru, .su и .рф доменов игнорируются значения полей NS-серверов, хеша NSS и домен регистрируется неделегированным. Для остальных зон не применим. Допустимые значения 0 и 1. |
| user\_servid | 32 | 32 | ID домена, задаваемый пользователем. Допустимые символы: цифры 0..9 и латинские буквы a..f. Автоматически идентификатор не создаётся, т.е. если он не был задан при создании услуги, то поле остаётся пустым. Необязательное поле. |
| comment | 0 | 255 | Комментарий. Необязательное поле. |
| admin\_comment | 0 | 255 | Административный комментарий. Необязательное поле. |
| pay\_type  ok\_if\_no\_money | — | | См. [Общие параметры оплаты](#inputparams_payment). |
| subtype | 0 | 15 | Тип регистрации. Опциональное поле. Возможное значение (кроме пустого значения по умолчанию): «preorder» — предзаказ доменов .РФ. |
| reg\_premium | 1 | | Регистрация премиум домена. Опциональное поле. Возможное значение (кроме пустого значения по умолчанию): «1» — показывает что клиент знает что это премиум домен и знает его цену. Цена премиум домена может быть от 10 до 1000 раз больше, чем обычная регистрация. Проверяйте цену на [/buy/domains/](/buy/domains/?query=example.com) или через функцию domain/get\_premium\_prices |

##### Описание полей для работы с папками

Поля этой категории являются необязательными.

| Параметр | Описание |
| --- | --- |
| folder\_name | Папка, в которую будет добавлен домен. Если указано имя несуществующей папки и не установлен флаг no\_new\_folder - папка будет создана. |
| folder\_id | Числовой идентификатор папки, в которую будет добавлен домен. |
| no\_new\_folder | Не создавать папку если она не существует. |

##### Описание полей контактных данных

###### Контактные данные для .RU/.SU/.РФ доменов

Обратите внимание, что в этой операции допустимо использовать один из двух взаимоисключающих наборов полей — данные организации (если домен регистрируется на организацию), либо данные частного лица (если домен регистрируется на частное лицо). Также некоторые поля могут быть многострочными.

##### Общие поля для .RU/.SU/.РФ

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| sms\_security\_number | 8 | 25 | Номер для отправки SMS-сообщений администратору домена.  Используется для уведомлений и подтверждений операций с доменом с целью обеспечения дополнительной безопасности и выполнения условий п. 5.2. [правил регистрации доменных имён в доменах .ru .рф](http://www.cctld.ru/ru/docs/rules.php). Запись не является обязательной.  *Пример: +7 927 1234567* |
| p\_addr\_zip | 2 | 15 | Почтовый индекс администратора домена.  *Пример: 101000* |
| p\_addr\_area | 3 | 65 | Область из почтового адреса администратора домена.  *Пример: Московская обл.* |
| p\_addr\_city | 3 | 25 | Название города из почтового адреса администратора домена.  *Пример: г. Москва* |
| p\_addr\_addr | 5 | 105 | Адрес включающий название улицы и номер дома (возможна доп. информация в виде номера корпуса/квартиры и т.д.).    *Пример: ул.Пупкина, 1, стр. 2, отдел мебели, офис 433* |
| p\_addr\_recipient | 5 | 45 | Имя получателя(физ лицо либо, название отдела)  *Пример: В. Лоханкин* |
| phone | 8 | 255 | Номер телефона администратора домена. Телефон указывается с международным кодом (включая символ +); международный код, код города и местный номер разделяются пробелами. Скобки и дефисы не допускаются.  *Пример: +7 495 8102233* |
| fax | 8 | 255 | Номер телефакса администратора домена. Номер телефакса указывается с международным кодом (включая символ +); международный код, код города и местный номер разделяются пробелами. Скобки и тире не допускаются. Запись не является обязательной. *Пример: +7 3432 811221* |
| e\_mail | 6 | 255 | Адрес электронной почты администратора домена в формате RFC-822.*Пример: ncc@test.ru* |

##### Данные организации (только при регистрации домена на организацию!)

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| org | 6 | 255 | Полное наименование организации-администратора домена латинскими буквами, предназначенное для использования услугой 'whois'.  *Пример1: Karla-Marla Uryupinsk State University*  *Пример2: "ROGA I KOPYTA", LTD.* |
| org\_r | 10 | 255 | Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке).  *Пример1: Урюпинский государственный университет имени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | 10 | 10 | Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |
| kpp | 9 | 9 | КПП организации (для Российских организаций). Обязательное поле.  *Пример: 632946014* |
| country | 2 | 2 | Двухбуквенный ISO-код страны, в которой зарегистрирована организация.  *Пример: RU* |
| address\_r\_zip | 2 | 15 | Почтовый индекс юридического адреса организации в соответствии с учредительными документами.  *Пример: 101000* |
| address\_r\_area | 3 | 65 | Область из юридического адреса организации в соответствии с учредительными документами.  *Пример: Московская обл.* |
| address\_r\_city | 3 | 25 | Название города юридического адреса организации в соответствии с учредительными документами.  *Пример: г. Москва* |
| address\_r\_addr | 5 | 105 | Адрес включающий название улицы и номер дома (возможна доп. информация в виде номера корпуса/квартиры и т.д.) юридического адреса организации в соответствии с учредительными документами.  *Пример: ул.Пупкина, 1, стр. 2* |

##### Данные частного лица (только при регистрации домена на частное лицо!)

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| person | 8 | 64 | Имя, первая буква отчества (без точки) и фамилия администратора домена, записанные латинскими буквами. Предназначено для использования услугой 'whois'. Для иностранцев поле содержит имя в оригинальном написании (или в английской транслитерации).  *Пример: Vassily N Pupkin* |
| person\_r\_surname | 1 | 32 | Фамилия администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит фамилию в оригинальном написании (при невозможности в английской транслитерации).  *Пример1: Пупкин*  *Пример2: Smith* |
| person\_r\_name | 1 | 32 | Имя администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации).  *Пример1: Василий*  *Пример2: John* |
| person\_r\_patronimic | 1 | 32 | Отчество администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит отчество в оригинальном написании (при невозможности в английской транслитерации) или остаётся пустым.  *Пример1: Николаевич*  *Пример2: I* |
| passport\_number | 6 | 30 | Серия и номер паспорта. В написании римских цифр допустимо использование только латинских букв. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются.  *Пример: 34 02 651241* |
| passport\_place | 3 | 200  много-  строчное | Наименование органа выдавшего паспорт. Запись может быть многострочной.  *Пример: 48 о/м г.Москвы* |
| passport\_date | 10 | 10 | Дата выдачи паспорта. Дата записывается в формате ДД.ММ.ГГГГ.  *Пример: 26.12.1992* |
| birth\_date | 10 | 10 | Дата рождения администратора домена в формате ДД.ММ.ГГГГ.  *Пример: 07.11.1917* |
| country | 2 | 2 | Двухбуквенный ISO-код страны, гражданином которой является частное лицо.  *Пример: RU* |

###### Контактные данные для доменной зоны .tj

Регистрация доменов по API в зоне .TJ не доступна.

На данный момент все контактные данные должны заполняться латиницей.

##### Данные владельца домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_type | 1 | 1 | Тип контакта. "1" для регистрации домена на физическиое лицо, "2" - для регистрации домена на юридическое лицо. |
| o\_whois | 2 | 64 | Описание домена. Отображается в whois-запросе. |
| o\_full\_name | 2 | 64 | Владелец домена: полное название организации или ФИО владельца. |
| o\_email | 6 | 90 | Адрес электронной почты владельца домена в формате RFC-822. |
| o\_phone | 10 | 16 | Телефон владельца домена, указывается в международном формате с пробелами между кодом страны, кодом города и внутренним номером.  *(Пример: +7 495 1234567 или +662 22 1234567)* |
| o\_fax | 10 | 16 | Факс владельца домена, указывается в международном формате с пробелами между кодом страны, кодом города и внутренним номером.  *(Пример: +7 495 1234567 или +662 22 1234567)* |
| o\_addr | 2 | 128 | Адрес владельца домена: юридический адрес организации в соответствии с учредительными документами или адрес проживания владельца домена. |
| o\_city | 2 | 64 | Адрес владельца домена: город. |
| o\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны владельца домена.  Список кодов всех стран можно найти [здесь](https://www.iso.org/obp/ui/#search) |

##### Данные администратора домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| a\_full\_name | 2 | 64 | ФИО администратора домена. |
| a\_nic\_name | 2 | 32 | Краткое имя администратора или его nickname, одно слово. |
| a\_email | 6 | 90 | Адрес электронной почты администратора домена в формате RFC-822. |
| a\_fax | 10 | 16 | Телефон(!) администратора домена, указывается в международном формате с пробелами между кодом страны, кодом города и внутренним номером.  *(Пример: +7 495 1234567 или +662 22 1234567)* |
| a\_addr | 2 | 128 | Адрес администратора домена. |
| a\_city |  |  | Адрес администратора домена: город. |
| a\_postcode | 3 | 10 | Адрес администратора домена: почтовый индекс. |
| a\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны администратора домена.  Список кодов всех стран можно найти [здесь](https://www.iso.org/obp/ui/#search) |

##### Данные технического администратора домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| t\_full\_name | 2 | 64 | ФИО технического администратора домена. |
| t\_nic\_name | 2 | 32 | Краткое имя технического администратора или его nickname, одно слово. |
| t\_email | 6 | 90 | Адрес электронной почты технического администратора домена в формате RFC-822. |
| t\_fax | 10 | 16 | Телефон(!) технического администратора домена, указывается в международном формате с пробелами между кодом страны, кодом города и внутренним номером.  *(Пример: +7 495 1234567 или +662 22 1234567)* |
| t\_addr | 2 | 128 | Адрес технического администратора домена. |
| t\_city | 2 | 64 | Адрес технического администратора домена: город. |
| t\_postcode | 3 | 10 | Адрес технического администратора домена: почтовый индекс. |
| t\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны технического администратора домена.  Список кодов всех стран можно найти [здесь](https://www.iso.org/obp/ui/#search) |

###### Регистрация доменов в зонах com.ua, kiev.ua

На данный момент все контактные данные должны заполняться латинскими буквами. Для доменных зон com.ua, kiev.ua используется только два вида контактов: контакты владельца (администратора) домена и технические контакты. Изменение контактных данных владельца домена после регистрации в автоматическом режиме невозможно.

##### Данные владельца домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_company | 5 | 80 | Название организации - владельца домена. |
| o\_first\_name | 2 | 40 | Имя контактного лица |
| o\_last\_name | 2 | 40 | Фамилия контактного лица |
| o\_email | 6 | 80 | Контактный email-адрес владельца домена. |
| o\_phone | 8 | 20 | Номер телефона контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| o\_fax | 8 | 20 | Номер телефакса контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| o\_addr | 8 | 80 | Адрес контактного лица: улица, дом, офис (квартира) |
| o\_city | 2 | 80 | Адрес контактного лица: город |
| o\_state | 2 | 40 | Адрес контактного лица: область/край/штат |
| o\_postcode | 3 | 10 | Почтовый индекс контактного лица |
| o\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны контактного лица. Список всех кодов стран можно найти [тут](https://www.iso.org/obp/ui/#search) |

##### Данные техподдержки домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| t\_company | 5 | 80 | Организация, осуществляющая техподдержку домена. |
| t\_first\_name | 2 | 40 | Имя контактного лица |
| t\_last\_name | 2 | 40 | Фамилия контактного лица |
| t\_email | 6 | 80 | Контактный email-адрес контактного лица. |
| t\_phone | 8 | 20 | Номер телефона контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| t\_fax | 8 | 20 | Номер телефакса контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| t\_addr | 8 | 80 | Адрес контактного лица: улица, дом, офис (квартира) |
| t\_city | 2 | 80 | Адрес контактного лица: город |
| t\_state | 2 | 40 | Адрес контактного лица: область/край/штат |
| t\_postcode | 3 | 10 | Почтовый индекс контактного лица |
| t\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны контактного лица. Список всех кодов стран можно найти [тут](https://www.iso.org/obp/ui/#search) |

###### Регистрация доменов в зоне pp.ua

На данный момент все контактные данные должны заполняться латинскими буквами. При регистрации домена в зоне pp.ua в контактных данных владельца домена необходимо указывать номер мобильного телефона. После регистрации домена на этот номер будет отправлено SMS с кодом активации домена, который вместе с именем домена нужно ввести здесь:

<http://www.pp.ua/rus/confirm.html>. Допускается регистрировать не более трёх доменов в месяц на один мобильный телефон. Подробнее с правилами домена pp.ua можно ознакомиться

[здесь](http://www.pp.ua/rus/policy.html). Остальные правила регистрации доменов в зоне pp.ua соответствуют правилам регистрации доменов в других зонах.

###### Регистрация доменов в зонах \*.kz (все домены Казахстана)

С 7 сентября 2010 г. в соответствии с [Правилами KazNIC](http://nic.kz/rules/) в контактных данных домена необходимо указывать адрес серверного оборудования, на котором располагается домен. Домены, зарегистрированные после указанной даты, обязаны располагаться на серверах Казахстана. Для указания адреса серверного оборудования при регистрации домена или изменении контактов необходимо добавить следующие поля в хеш contacts.

##### Адрес серверного оборудования

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| srvloc\_state | 2 | 40 | Область / Штат |
| srvloc\_city | 2 | 40 | Город |
| srvloc\_street | 2 | 255 | Адрес (улица, дом) |

##### Описание домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| reg\_purpose | 5 | 150 | Описание домена |

Если не планируется сразу привязывать домен к хостингу или парковке, можно использовать следующие значения для полей адреса серверного оборудования:

|  |  |
| --- | --- |
| srvloc\_state | KAR |
| srvloc\_city | Karaganda |
| srvloc\_street | Chizhevskogo, 17 |
| reg\_purpose | Private domain |

###### Регистрация доменов в зоне .es

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_es\_form\_juridica | 1 | 3 | Форма собственности владельца домена  **1** - Физическое лицо  **39** - Объединение с экономической целью  **47** - Объединение  **59** - Спортивное общество  **68** - Ассоциация  **124** - Сберегательный банк  **150** - Общее имущество (супругов)  **152** - Кондоминиум  **164** - Монашеский орден или религиозная организация  **181** - Консульство  **197** - Ассоциация по вопросам государственного права  **203** - Посольство  **229** - Муниципалитет  **269** - Спортивная федерация  **286** - Фонд  **365** - Общество взаимного страхования  **434** - Орган правительства провинции  **436** - Орган центрального правительства  **439** - Политическая партия  **476** - Профессиональный союз  **510** - Фермерское товарищество  **524** - Акционерная компания открытого типа с ограниченной ответственностью/корпорация  **525** - Спортивная акционерная компания открытого типа с ограниченной ответственностью  **554** - Товарищество  **560** - Полное товарищество  **562** - Товарищество с ограниченной ответственностью  **566** - Кооператив  **608** - Компания, принадлежащая сотрудникам  **612** - Общество с ограниченной ответственностью  **713** - Испанский филиал компании  **717** - Консорциум/совместное предприятие  **744** - Компания с ограниченной ответственностью, принадлежащая сотрудникам  **745** - Государственная организация уровня провинции  **746** - Государственная организация национального уровня  **747** - Государственная организация местного уровня  **877** - Прочее  **878** - Надзорный совет по вопросам наименования места происхождения товара  **879** - Организация, управляющая природной территорией |
| a\_es\_form\_juridica | 1 | 3 | Форма собственности администратора домена  **1** - Физическое лицо |
| b\_es\_form\_juridica | 1 | 3 | Форма собственности биллинга домена  **1** - Физическое лицо |
| t\_es\_form\_juridica | 1 | 3 | Форма собственности технической поддержки домена  **1** - Физическое лицо |
| o\_es\_identificacion | 3 | 30 | Номер удостоверяющего документа владельца домена |
| a\_es\_identificacion | 3 | 30 | Номер удостоверяющего документа администратора домена |
| b\_es\_identificacion | 3 | 30 | Номер удостоверяющего документа биллинга домена |
| t\_es\_identificacion | 3 | 30 | Номер удостоверяющего документа технической поддержки домена |
| o\_es\_tipo\_identificacion | 1 | 1 | Вид удостоверяющего документа владельца домена  **1** - DNI or NIF  **3** - NIE  **0** - Other ID |
| a\_es\_tipo\_identificacion | 1 | 1 | Вид удостоверяющего документа администратора домена  **1** - DNI or NIF  **3** - NIE  **0** - Other ID |
| b\_es\_tipo\_identificacion | 1 | 1 | Вид удостоверяющего документа биллинга домена  **1** - DNI or NIF  **3** - NIE  **0** - Other ID |
| t\_es\_tipo\_identificacion | 1 | 1 | Вид удостоверяющего документа технической поддержки домена  **1** - DNI or NIF  **3** - NIE  **0** - Other ID |

###### Регистрация доменов в зоне .jobs

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| company\_url | 4 | 255 | URL сайта компании |
| ind\_classification | 1 | 2 | Классификация отрасли  **2** - "Accounting/Banking/Finance (2)"  **3** - "Agriculture/Farming (3)"  **21** - "Biotechnology/Science (21)"  **5** - "Computer/Information Technology (5)"  **4** - "Construction/Building Services (4)"  **12** - "Consulting (12)"  **6** - "Education/Training/Library (6)"  **7** - "Entertainment (7)"  **13** - "Environmental (13)"  **19** - "Hospitality (19)"  **10** - "Government/Civil Service (10)"  **11** - "Healthcare (11)"  **15** - "HR/Recruiting (15)"  **16** - "Insurance (16)"  **17** - "Legal (17)"  **18** - "Manufacturing (18)"  **20** - "Media/Advertising (20)"  **9** - "Parks & Recreation (9)"  **26** - "Pharmaceutical (26)"  **22** - "Real Estate (22)"  **14** - "Restaurant/Food Service (14)"  **23** - "Retail (23)"  **8** - "Telemarketing (8)"  **9** - "Transportation (24)"  **25** - "Other (25)" |
| hra\_name | 2 | 3 | Является ли организация членом Human Resource Association?  **no** - Нет  **yes** - Да |
| job\_title | 4 | 255 | Должность контактного лица |
| admin\_type | 1 | 1 | Является ли контакт административным  **0** - Нет  **1** - Да |

###### Регистрация доменов в зоне .us

| Параметр | Максимальная длина поля | Описание |
| --- | --- | --- |
| RselnexusAppPurpose | 2 | Сфера использования домена  **P1** - Бизнес, для получения прибыли  **P2** - Бизнес, не для получением прибыли  **P3** - Для персонального использования  **P4** - Для образовательных целей  **P5** - Для государственных целей |
| RselnexusCategory | 3 | Владельец домена  **C11** - Физическое лицо - Гражданин США  **C12** - Физическое лицо - постоянный резидент США или любой из его территорий  **C21** - Юридическое лицо или организация, инкорпорированная в одном из 50-ти штатов США  **C31** - Юридическое лицо или организация, которую регулярно ведет законную деятельность в США  **C32** - Юридическое лицо или организация, которая имеет офис или другое имущество в США |

###### Регистрация доменов в зонах .МОСКВА, .MOSCOW, .ДЕТИ, .TATAR, .RU.NET, .COM.RU, .MSK.RU, .SPB.RU и других геодоменов для организаций

В зонах .RU.NET, .COM.RU, .EXNET.SU и геодоменах .ABKHAZIA.SU, .ADYGEYA.RU, .ADYGEYA.SU, .AKTYUBINSK.SU, .ARKHANGELSK.SU, .ARMENIA.SU, .ASHGABAD.SU, .AZERBAIJAN.SU, .BALASHOV.SU, .BASHKIRIA.RU, .BASHKIRIA.SU, .BIR.RU, .BRYANSK.SU, .BUKHARA.SU, .CBG.RU, .CHIMKENT.SU, .DAGESTAN.RU, .DAGESTAN.SU, .EAST-KAZAKHSTAN.SU, .EXNET.SU, .GEORGIA.SU, .GROZNY.RU, .GROZNY.SU, .IVANOVO.SU, .JAMBYL.SU, .KALMYKIA.RU, .KALMYKIA.SU, .KALUGA.SU, .KARACOL.SU, .KARAGANDA.SU, .KARELIA.SU, .KHAKASSIA.SU, .KRASNODAR.SU, .KURGAN.SU, .KUSTANAI.RU, .KUSTANAI.SU, .LENUG.SU, .MANGYSHLAK.SU, .MARINE.RU, .MORDOVIA.RU, .MORDOVIA.SU, .MSK.RU, .MSK.SU, .MURMANSK.SU, .MYTIS.RU, .NALCHIK.RU, .NALCHIK.SU, .NAVOI.SU, .NORTH-KAZAKHSTAN.SU, .NOV.RU, .NOV.SU, .OBNINSK.SU, .PENZA.SU, .POKROVSK.SU, .PYATIGORSK.RU, .RU.NET, .SOCHI.SU, .SPB.RU, .SPB.SU, .TASHKENT.SU, .TERMEZ.SU, .TOGLIATTI.SU, .TROITSK.SU, .TSELINOGRAD.SU, .TULA.SU, .TUVA.SU, VLADIKAVKAZ.RU, .VLADIKAVKAZ.SU, .VLADIMIR.RU, .VLADIMIR.SU, .VOLOGDA.SU надо указывать только владельца домена, без администартора и техподдержки

###### Данные владельца домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| o\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| o\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| o\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| o\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| o\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| o\_city | 2 | 40 | Город.  *Пример: Moscow* |
| o\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| o\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| o\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| o\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| o\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| o\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| o\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| o\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| o\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| o\_company | 5 | 80 | Компания.  *Пример: Novye vremena, CJSC* |
| o\_company\_ru | 5 | 255 | Компания (по-русски).  *Пример: ООО Новые времена* |
| o\_code | 6 | 12 | ИНН организации.  *Пример: 1234567890* |
| o\_l\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| o\_l\_addr | 10 | 255 | Адрес.  *Пример: Koshkina str, 15-4* |
| o\_l\_city | 2 | 40 | Город.  *Пример: Moscow* |
| o\_l\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| o\_l\_postcode | 3 | 8 | Почтовый Индекс.  *Пример: 123456* |
| o\_l\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| o\_l\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |

###### Данные администратора домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| a\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| a\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| a\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| a\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| a\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| a\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| a\_city | 2 | 40 | Город.  *Пример: Moscow* |
| a\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| a\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| a\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| a\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| a\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| a\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| a\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| a\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| a\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| a\_company | 5 | 80 | Компания.  *Пример: Novye vremena, CJSC* |
| a\_company\_ru | 5 | 255 | Компания (по-русски).  *Пример: ООО Новые времена* |
| a\_code | 6 | 12 | ИНН организации.  *Пример: 1234567890* |
| a\_l\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| a\_l\_addr | 10 | 255 | Адрес.  *Пример: Koshkina str, 15-4* |
| a\_l\_city | 2 | 40 | Город.  *Пример: Moscow* |
| a\_l\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| a\_l\_postcode | 3 | 8 | Почтовый Индекс.  *Пример: 123456* |
| a\_l\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| a\_l\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |

###### Данные техподдержки домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| t\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| t\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| t\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| t\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| t\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| t\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| t\_city | 2 | 40 | Город.  *Пример: Moscow* |
| t\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| t\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| t\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| t\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| t\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| t\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| t\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| t\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| t\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| t\_company | 5 | 80 | Компания.  *Пример: Novye vremena, CJSC* |
| t\_company\_ru | 5 | 255 | Компания (по-русски).  *Пример: ООО Новые времена* |
| t\_code | 6 | 12 | ИНН организации.  *Пример: 1234567890* |
| t\_l\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| t\_l\_addr | 10 | 255 | Адрес.  *Пример: Koshkina str, 15-4* |
| t\_l\_city | 2 | 40 | Город.  *Пример: Moscow* |
| t\_l\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| t\_l\_postcode | 3 | 8 | Почтовый Индекс.  *Пример: 123456* |
| t\_l\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| t\_l\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |

###### Регистрация доменов в зонах .МОСКВА, .MOSCOW, .ДЕТИ, .TATAR, .RU.NET, .COM.RU, .MSK.RU, SPB.RU и других геодоменов для частных лиц

В зонах .RU.NET, .COM.RU, .EXNET.SU и геодоменах .ABKHAZIA.SU, .ADYGEYA.RU, .ADYGEYA.SU, .AKTYUBINSK.SU, .ARKHANGELSK.SU, .ARMENIA.SU, .ASHGABAD.SU, .AZERBAIJAN.SU, .BALASHOV.SU, .BASHKIRIA.RU, .BASHKIRIA.SU, .BIR.RU, .BRYANSK.SU, .BUKHARA.SU, .CBG.RU, .CHIMKENT.SU, .DAGESTAN.RU, .DAGESTAN.SU, .EAST-KAZAKHSTAN.SU, .EXNET.SU, .GEORGIA.SU, .GROZNY.RU, .GROZNY.SU, .IVANOVO.SU, .JAMBYL.SU, .KALMYKIA.RU, .KALMYKIA.SU, .KALUGA.SU, .KARACOL.SU, .KARAGANDA.SU, .KARELIA.SU, .KHAKASSIA.SU, .KRASNODAR.SU, .KURGAN.SU, .KUSTANAI.RU, .KUSTANAI.SU, .LENUG.SU, .MANGYSHLAK.SU, .MARINE.RU, .MORDOVIA.RU, .MORDOVIA.SU, .MSK.RU, .MSK.SU, .MURMANSK.SU, .MYTIS.RU, .NALCHIK.RU, .NALCHIK.SU, .NAVOI.SU, .NORTH-KAZAKHSTAN.SU, .NOV.RU, .NOV.SU, .OBNINSK.SU, .PENZA.SU, .POKROVSK.SU, .PYATIGORSK.RU, .RU.NET, .SOCHI.SU, .SPB.RU, .SPB.SU, .TASHKENT.SU, .TERMEZ.SU, .TOGLIATTI.SU, .TROITSK.SU, .TSELINOGRAD.SU, .TULA.SU, .TUVA.SU, VLADIKAVKAZ.RU, .VLADIKAVKAZ.SU, .VLADIMIR.RU, .VLADIMIR.SU, .VOLOGDA.SU надо указывать только владельца домена, без администартора и техподдержки

###### Данные владельца домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| o\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| o\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| o\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| o\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| o\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| o\_city | 2 | 40 | Город.  *Пример: Moscow* |
| o\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| o\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| o\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| o\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| o\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| o\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| o\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| o\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| o\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| o\_code | 6 | 12 | ИНН.  *Пример: 1234567890* |
| o\_birth\_date | 10 | 10 | Дата рождения.  *Пример: 08.05.1965* |
| o\_passport\_date | 10 | 30 | Дата выдачи паспорта.  *Пример: 30.01.1992* |
| o\_passport\_number | 6 | 30 | Серия и номер паспорта.  *Пример: 12 34 567890* |
| o\_passport\_place | 10 | 200 | Место выдачи паспорта.  *Пример: выдан 123 отделением милиции г.Москвы* |

###### Данные администратора домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| a\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| a\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| a\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| a\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| a\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| a\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| a\_city | 2 | 40 | Город.  *Пример: Moscow* |
| a\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| a\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| a\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| a\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| a\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| a\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| a\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| a\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| a\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| a\_code | 6 | 12 | ИНН.  *Пример: 1234567890* |
| a\_birth\_date | 10 | 10 | Дата рождения.  *Пример: 08.05.1965* |
| a\_passport\_date | 10 | 30 | Дата выдачи паспорта.  *Пример: 30.01.1992* |
| a\_passport\_number | 6 | 30 | Серия и номер паспорта.  *Пример: 12 34 567890* |
| a\_passport\_place | 10 | 200 | Место выдачи паспорта.  *Пример: выдан 123 отделением милиции г.Москвы* |

###### Данные техподдержки домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| t\_first\_name | 2 | 40 | Имя.  *Пример: Ivan* |
| t\_first\_name\_ru | 2 | 40 | Имя (по-русски).  *Пример: Иван* |
| t\_last\_name | 2 | 40 | Фамилия.  *Пример: Sidorov* |
| t\_last\_name\_ru | 2 | 40 | Фамилия (по-русски).  *Пример: Иванов* |
| t\_patronimic | 2 | 25 | Отчество.  *Пример: Sidorovich* |
| t\_patronimic\_ru | 2 | 25 | Отчество (по-русски).  *Пример: Иванович* |
| t\_city | 2 | 40 | Город.  *Пример: Moscow* |
| t\_city\_ru | 3 | 80 | Город (по-русски).  *Пример: г. Москва* |
| t\_addr | 8 | 80 | Адрес.  *Пример: Koshkina str, 15-4* |
| t\_addr\_ru | 10 | 255 | Адрес (по-русски).  *Пример: ул. Кошкина, д.15, кв.4* |
| t\_state | 2 | 40 | Регион.  *Пример: Moscow* |
| t\_state\_ru | 2 | 40 | Регион (по-русски).  *Пример: Московская обл.* |
| t\_phone | 8 | 20 | Телефон.  *Пример: +7 495 1234567  +7 495 1234569* |
| t\_email | 6 | 90 | EMail.  *Пример: adm-group@newtime.ru  sidor@newtime.msk.su* |
| t\_postcode | 3 | 10 | Почтовый Индекс.  *Пример: 123456* |
| t\_country\_code | 2 | 2 | Страна.  *Пример: RU* |
| t\_code | 6 | 12 | ИНН.  *Пример: 1234567890* |
| t\_birth\_date | 10 | 10 | Дата рождения.  *Пример: 08.05.1965* |
| t\_passport\_date | 10 | 30 | Дата выдачи паспорта.  *Пример: 30.01.1992* |
| t\_passport\_number | 6 | 30 | Серия и номер паспорта.  *Пример: 12 34 567890* |
| t\_passport\_place | 10 | 200 | Место выдачи паспорта.  *Пример: выдан 123 отделением милиции г.Москвы* |

###### Регистрация доменов в других зонах

##### Данные владельца домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| o\_company | 5 | 80 | Название организации - владельца домена. |
| o\_first\_name | 2 | 40 | Имя контактного лица |
| o\_last\_name | 2 | 40 | Фамилия контактного лица |
| o\_email | 6 | 90 | Контактный email-адрес владельца домена. |
| o\_phone | 8 | 20 | Номер телефона владельца домена. Телефон указывается в международном формате.  *(Пример: +7.4952171179).* |
| o\_fax | 8 | 20 | Номер факса владельца домена. Номер указывается в международном формате. Необязательное поле.  *(Пример: +7.4952171179).* |
| o\_addr | 8 | 80 | Адрес владельца домена: улица, дом, офис (квартира) |
| o\_city | 2 | 80 | Адрес владельца домена: город |
| o\_state | 2 | 40 | Адрес владельца домена: область/край/штат |
| o\_postcode | 3 | 10 | Почтовый индекс владельца домена |
| o\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны владельца домена. Некоторые доменные зоны допускают указание только стран официально подпадающих под эту зону. Например, для .eu допустимо указывать только страны входящие в EC. Список кодов всех стран можно найти [здесь](https://www.iso.org/obp/ui/#search) |

##### Данные администратора домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| a\_company | 5 | 80 | Название организации - владельца домена. |
| a\_first\_name | 2 | 40 | Имя контактного лица |
| a\_last\_name | 2 | 40 | Фамилия контактного лица |
| a\_email | 6 | 80 | Контактный email-адрес владельца домена. |
| a\_phone | 8 | 20 | Номер телефона контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| a\_fax | 8 | 20 | Номер телефакса контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| a\_addr | 8 | 80 | Адрес контактного лица: улица, дом, офис (квартира) |
| a\_city | 2 | 80 | Адрес контактного лица: город |
| a\_state | 2 | 40 | Адрес контактного лица: область/край/штат |
| a\_postcode | 3 | 10 | Почтовый индекс контактного лица |
| a\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны контактного лица. Список всех кодов стран можно найти [тут](https://www.iso.org/obp/ui/#search) |

##### Данные техподдержки домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| t\_company | 5 | 80 | Организация, осуществляющая техподдержку домена. |
| t\_first\_name | 2 | 40 | Имя контактного лица |
| t\_last\_name | 2 | 40 | Фамилия контактного лица |
| t\_email | 6 | 80 | Контактный email-адрес контактного лица. |
| t\_phone | 8 | 20 | Номер телефона контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| t\_fax | 8 | 20 | Номер телефакса контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| t\_addr | 8 | 80 | Адрес контактного лица: улица, дом, офис (квартира) |
| t\_city | 2 | 80 | Адрес контактного лица: город |
| t\_state | 2 | 40 | Адрес контактного лица: область/край/штат |
| t\_postcode | 3 | 10 | Почтовый индекс контактного лица |
| t\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны контактного лица. Список всех кодов стран можно найти [тут](https://www.iso.org/obp/ui/#search) |

###### Биллинговые контакты домена

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| b\_company | 5 | 80 | Организация. |
| b\_first\_name | 2 | 40 | Имя контактного лица |
| b\_last\_name | 2 | 40 | Фамилия контактного лица |
| b\_email | 6 | 80 | Контактный email-адрес контактного лица. |
| b\_phone | 8 | 20 | Номер телефона контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| b\_fax | 8 | 20 | Номер телефакса контактного лица. Телефон указывается с международном формате.  *(Пример: +7.4952171179).* |
| b\_addr | 8 | 80 | Адрес контактного лица: улица, дом, офис (квартира) |
| b\_city | 2 | 80 | Адрес контактного лица: город |
| b\_state | 2 | 40 | Адрес контактного лица: область/край/штат |
| b\_postcode | 3 | 10 | Почтовый индекс контактного лица |
| b\_country\_code | 2 | 2 | Двухбуквенный ISO-код страны контактного лица. Список всех кодов стран можно найти [тут](https://www.iso.org/obp/ui/#search) |

##### Дополнительные данные для доменов в зонах .COM, .NET, .ORG, .ASIA, .BIZ, .NAME, .INFO, .MOBI, .UK, .CC, .TV, .WS, .BZ, .ME

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| private\_person\_flag | 1 | 1 | Активация услуги Privacy Protection |

##### Дополнительные данные для доменов в зоне .US

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| RselnexusAppPurpose | 2 | 2 | Сфера использования домена Возможные значения:  **P1** - Бизнес, для получения прибыли  **P2** - Бизнес, не для получением прибыли  **P3** - Для персонального использования  **P4** - Для образовательных целей  **P5** - Для государственных целей |
| RselnexusCategory | 3 | 3 | Владелец домена: Возможные значения:  **C11** - Физическое лицо - Гражданин США  **C12** - Физическое лицо - постоянный резидент США или любой из его территорий  **C21** - Юридическое лицо или организация, инкорпорированная в одном из 50-ти штатов США  **C31** - Юридическое лицо или организация, которую регулярно ведет законную деятельность в США  **C32** - Юридическое лицо или организация, которая имеет офис или другое имущество в США |

##### Дополнительные данные для доменов в зоне .PRO

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| pro\_profession | 1 | 100 | Профессия владельца домена Возможные значения:   * Acupuncturists * Allied Health Professionals * Ambulance Services * Architects * Asbestos Removal Professionals * Barbers and Barber Shops * Certified Financial Analysts * Certified Financial Planners * Certified Public Accountants * Check Cashers * Chiropractors * Contractors, Home Improvement * Cosmetologists and Aestheticians * Debt Collectors * Dentists and Dental Hygienists * Dieticians and Nutritionists * Doctors * Educators * Electricians * Electrologists * Emergency Medical Technician * Engineers and Land Surveyors * Finance Companies * Financial Professional * Funeral Services * Health Care * Hearing Instrument Specialists * Home Inspectors * HVAC Technicians * Insurance * Investment Advisors * Landscape Architects * Lawyers * Lead Paint Inspectors * Manufactured Building Producers * Massage Therapy * Money Transmitters * Mortgage Lenders and Brokers * Municipal Building Inspectors * Nurses and Nurse Aides * Nursing Home Administrators * Nutritionists * Opticians * Optometrists * Perfusionist * Pharmacists * Physical Therapists * Physician Assistants * Physicians * Plumbers and Gas Fitters * Podiatrists * Psychologists * Public Relations * Radio and TV Technicians * Real Estate * Real Estate Appraisers * Respiratory Therapists * Sanitarians * Social Workers * Speech Pathologists and Audiologists * Veterinarians * Water Plant Operator * X-Ray Technicians * Internet Professional * Medical Professional * Legal Professional * Other |

##### Дополнительные данные для доменов в зоне .LV или .RO

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| idnumber | 3 | 100 | Единый регистрационнный номер |
| vatid | 3 | 100 | VAT ID (ИНН) |
| registrant\_type | 10 | 12 | Тип регистратора. Возможные значения:  **individual**  **organization** |

##### Дополнительные данные для доменов в зонах .САЙТ и .ОНЛАЙН

Дополнительные поля заполняются только знаками кириллического алфавита, указываются данные администратора и владельца домена, контакты технического специалиста и специалиста по биллингу.

Тип контакта определяется префиксом: a, t, o, b

Набор полей ( на примере административного контакта )

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| a\_first\_name\_r | 2 | 25 | Имя контактного лица |
| a\_last\_name\_r | 2 | 25 | Фамилия контактного лица |
| a\_company\_r | 5 | 255 | Организация |
| a\_city\_r | 3 | 80 | Город |
| a\_addr\_r | 15 | 255 | Адрес |

###### DNS-серверы домена

Для регистрации домена должно быть указано не менее двух серверов. В случае указания NS-серверов на базе одного из заказываемых доменов, обязателельно должны быть указаны IP-адреса этих NS-серверов.

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| ns0 | 6 | 80 | Имя хоста первого DNS-сервера. |
| ns1 | 6 | 80 | Имя хоста второго DNS-сервера. |
| ns2 | 6 | 80 | Имя хоста третьего DNS-сервера. |
| ns3 | 6 | 80 | Имя хоста четвертого DNS-сервера. |
| ns0ip | 8 | 15 | IP-адрес первого DNS-сервера. Необязательное поле. Используется, только если имя DNS-сервера содержит имя регистрируемого домена. |
| ns1ip | 8 | 15 | IP-адрес второго DNS-сервера. Необязательное поле. Используется, только если имя DNS-сервера содержит имя регистрируемого домена. |
| ns2ip | 8 | 15 | IP-адрес третьего DNS-сервера. Необязательное поле. Используется, только если имя DNS-сервера содержит имя регистрируемого домена. |
| ns3ip | 8 | 15 | IP-адрес четвертого DNS-сервера. Необязательное поле. Используется, только если имя DNS-сервера содержит имя регистрируемого домена. |

Примечание: Для поддержки DNS могут быть бесплатно использованы сервера REG.RU. Для этого в качестве DNS-серверов необходимо указать сервера ns1.reg.ru и ns2.reg.ru. При этом на данных серверах будет прописана зона для Вашего домена. Управлять зоной впоследствии можно будет через web-интерфейс сайта reg.ru.

###### Поддержка обработки списка услуг:

Только для VIP клиентов.

###### Поддержка unicode-доменов

Помимо поддержки русского языка в кириллических зонах .РФ, .РУС, .САЙТ, .ОНЛАЙН, .МОСКВА и других, имеется поддержка других языков и символов в основных зонах.

Зона .SU поддерживает армянский, греческий, грузинский и другие языки, полный список всех символов доступен на сайте [реестра](https://www.tcinet.ru/su/ver2_unicode.php).

Зоны .САЙТ и .ОНЛАЙН поддерживают символы украинского, белорусского и болгарского языков.

Зоны домена .UA поддерживают не только украинский алфавит, но и другие. Полный список символов большинства .UA зон есть на [cctld ua](https://hostmaster.ua/idn/) и [uanic](http://uanic.net/tablicya-dozvolenix-simvoliv/).

Зоны доменов, поддерживаемых реестром Verisign, такие как .COM, .NET и другие, могут иметь названия на многих европейских языках, кириллице, китайских, корейских, японских иероглифах, арабском, тайском и других. Полный список смотрите [тут](https://www.verisign.com/assets/idn-valid-language-tags.pdf).

Каждая из зон, находящихся на обслуживании реестра CentralNic, такие как .RU.COM, .XYZ, .SITE, .ONLINE и другие, имеет свой набор поддерживаемых языков. Их полный список можно посмотреть на сайте [реестра](https://www.centralnic.com/support/idn-tables). При выборе языка отобразится список зон, которые его поддерживают.

Поддерживаемые языки зон .INFO, .ORG и других можно посмотреть на сайте [IANA тут](https://www.iana.org/domains/idn-tables).

Запросы на регистрацию доменов можно отправлять как в unicode, так и в punycode. Если какой-то символ в имени домена наша система не принимает, сверьтесь с реестром перед написанием обращения в техническую поддержку.

##### Поля ответа:

| Поле | Описание |
| --- | --- |
| bill\_id | Номер счёта, созданного по запросу. |
| payment | Сумма заказа в рублях. |
| pay\_type | Способ оплаты. На данный момент возможна только предоплата, prepay. |
| pay\_notes | Комментарий, относящийся к используемому способу оплаты. |
| domains | Список доменов с результатом, содержит поля: dname — имя домена, result — поле результатов, service\_id — внутреннний id домена в случае успешного принятия заявки. Поле result может иметь следующие значения:  success — заказ на регистрацию принят,  Invalid TLD — ошибка в имени доменной зоны,  Registraion in this TLD unavailable — регистрация доменов этой зоны ещё недоступна,  Invalid punycode input — ошибка в punycode имени домена,  Domain\_name is invalid or unsupported zone — ошибка в доменном имени или неподдерживаемая зона,  Unavailable domain name — такое доменное имя не доступно для регистрации. |

#### Примеры запросов:

пример заказа одного .ru домена, используя запрос в «PLAIN» формате:

```
https://api.reg.ru/api/regru2/domain/create
birth_date=01.01.2000
country=RU
descr=Vschizh site
domain_name=vschizh.ru
e_mail=test@test.ru
ns0=ns1.reg.ru
ns1=ns2.reg.ru
output_content_type=plain
p_addr_addr=ул. Княжеска, д.1
p_addr_area=
p_addr_city=г. Вщиж
p_addr_recipient=Рюрику Святославу Владимировичу
p_addr_zip=123456
passport_date=01.01.1164
passport_number=22 44 668800
passport_place=выдан по месту правления
password=test
person=Svyatoslav V Ryurik
person_r_name=Святослав
person_r_patronimic=Владимирович
person_r_surname=Рюрик
phone=+7 495 1234567
username=test
```

Обратите внимание, что если Ваша библиотека/программа не делает полного автоматического преобразования данных, то «+» надо передавать как «%2B». пример подобного запроса в JSON-формате, для наглядности отправляемые данные сначала представлены ввиде структуры на Perl-e с преобразованием её в JSON-формат:

```
$jsondata = {
    contacts => {
        descr => 'Vschizh site',
        person => 'Svyatoslav V Ryurik',
        person_r_surname => 'Рюрик',
        person_r_name => 'Святослав',
        person_r_patronimic => 'Владимирович',
        passport_number => '22 44 668800',
        passport_place => 'выдан по месту правления',
        passport_date => '01.09.1164',
        birth_date => '01.01.2000',
        p_addr_zip => '123456',
        p_addr_area => '',
        p_addr_city => 'г. Вщиж',
        p_addr_addr => 'ул. Княжеска, д.1',
        p_addr_recipient => 'Рюрику Святославу Владимировичу',
        phone => '+7 495 1234567',
        e_mail => 'test@test.ru',
        country => 'RU',
    },
    nss => {
        ns0 => 'ns1.reg.ru',
        ns1 => 'ns2.reg.ru',
    },
    domain_name => 'vschizh.su',
};

$jsondata = JSO::XS->new->utf8->encode( $jsondata );
```

и сам запрос:

```
https://api.reg.ru/api/regru2/domain/create
input_data={"contacts":{"country":"RU","e_mail":"test@test.ru","person_r_surname":"Рюрик","person_r_name":"Святослав","person_r_patronimic":"Владимирович","phone":"%2B7 495 1234567","birth_date":"01.01.2000","descr":"Vschizh site","person":"Svyatoslav V Ryurik","p_addr_zip":"123456","p_addr_area":"","p_addr_city":"г. Вщиж","p_addr_addr":"ул. Княжеска, д.1","p_addr_recipient":"Рюрику Святославу Владимировичу","passport_number":"22 44 668800","passport_place":"выдан по месту правления","passport_date":"01.09.1164"},"domain_name":"vschizh.su","nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"}}
input_format=json
output_content_type=plain
password=test
username=test
```

то же самое, но предполагается, что контактные данные хранятся в профиле my\_like\_ru\_profile:

```
$jsondata = {
    profile_type => 'RU.PP',
    profile_name => 'my_like_ru_profile',
    nss => {
        ns0 => 'ns1.reg.ru',
        ns1 => 'ns2.reg.ru',
    },
    domain_name => 'vschizh.su',
};

$jsondata = JSON::XS->new->utf8->encode( $jsondata );
```

и сам запрос:

```
https://api.reg.ru/api/regru2/domain/create
input_data={"profile_type":"RU.PP","profile_name":"my_like_ru_profile","domain_name":"vschizh.su","nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"}}
input_format=json
output_content_type=plain
password=test
username=test
```

первый вариант, но оптовая регистрация:

```
$jsondata = {
    contacts => {
        descr => 'Vschizh site',
        person => 'Svyatoslav V Ryurik',
        person_r_surname => 'Рюрик',
        person_r_name => 'Святослав',
        person_r_patronimic => 'Владимирович',
        passport_number => '22 44 668800',
        passport_place => 'выдан по месту правления',
        passport_date => '01.09.1164',
        birth_date => '01.01.2000',
        p_addr_zip => '123456',
        p_addr_area => ''
        p_addr_city => 'г. Вщиж'
        p_addr_addr => 'ул. Княжеска, д.1'
        p_addr_recipient => 'Рюрику Святославу Владимировичу'
        phone => '+7 495 8102233',
        e_mail => 'test@test.ru',
        country => 'RU',
    },
    nss => {
        ns0 => 'ns1.reg.ru',
        ns1 => 'ns2.reg.ru',
    },
    domains => [
        { dname => 'vschizh.ru', srv_certificate => 'free', srv_parking => 'free' },
        { dname => 'vschizh.su', srv_webfwd => '' },
    ],
};

$jsondata = JSON::XS->new->utf8->encode( $jsondata );
```

и сам запрос:

```
https://api.reg.ru/api/regru2/domain/create
input_data={"contacts":{"country":"RU","e_mail":"ruriksvyatvlad@mail.ru","person_r_surname":"Рюрик","person_r_name":"Святослав","person_r_patronimic":"Владимирович","phone":"+79142345678","birth_date":"01.01.2000","descr":"Vschizh site","person":"Svyatoslav V Ryurik","p_addr_zip":"123456","p_addr_area":"","p_addr_city":"г. Вщиж","p_addr_addr":"ул. Княжеска, д.1","p_addr_recipient":"Рюрику Святославу Владимировичу","passport_number":"22 44 668800","passport_place":"выдан по месту правления","passport_date":"01.09.1164"},"domains":[{"dname":"vschizh.su","srv_certificate":"free","srv_parking":"free"},{"dname":"vschizh.su","srv_webfwd":""}],"nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"}}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример запроса на регистрацию доменов в зоне .МОСКВА и .MOSCOW:

```
{
   "input_format" : "json",
   "username" : "test",
   "password" : "test",
   "input_data" : {
      "output_content_type" : "plain",
      "io_encoding" : "utf8",
      "show_input_params" : 1,
      "lang" : "en",
      "output_format" : "json",
      "contacts" : {
         "o_first_name_ru" : "Иван",
         "a_first_name_ru" : "Иван",
         "t_code" : "6663058366",
         "t_first_name" : "Ivan",
         "o_patronimic" : "Ivanovich",
         "t_phone" : "+7.953535385",
         "t_first_name_ru" : "Иван",
         "o_l_state" : "Moscow",
         "t_city" : "Moscow",
         "o_state" : "Moscow",
         "a_last_name_ru" : "Иванов",
         "o_addr" : "Bolshoy sobachiy, 33",
         "a_addr_ru" : "Большой собачий переулок, 33",
         "o_l_state_ru" : "Москва",
         "o_company_ru" : "ООО \"Лунтик\"",
         "t_patronimic" : "Ivanovich",
         "t_last_name_ru" : "Иванов",
         "a_l_city_ru" : "Москва",
         "a_city_ru" : "Москва",
         "o_l_city" : "Moscow",
         "o_first_name" : "Ivan",
         "a_state" : "Moscow",
         "o_l_city_ru" : "Москва",
         "t_country_code" : "RU",
         "t_addr_ru" : "Большой собачий переулок, 33",
         "a_company_ru" : "ООО \"Лунтик\"",
         "t_email" : "srs-devel@reg.ru",
         "a_l_state" : "Moscow",
         "o_patronimic_ru" : "Иванович",
         "t_company_ru" : "ООО \"Лунтик\"",
         "o_code" : "6663058366",
         "a_l_city" : "Moscow",
         "a_l_addr" : "Bolshoy sobachiy, 33",
         "t_patronimic_ru" : "Иванович",
         "o_phone" : "+7.953535385",
         "a_company" : "Luntik, LLC",
         "o_email" : "srs-devel@reg.ru",
         "o_addr_ru" : "Большой собачий переулок, 33",
         "o_last_name_ru" : "Иванов",
         "t_l_postcode" : "191000",
         "a_postcode" : "191000",
         "a_addr" : "Bolshoy sobachiy, 33",
         "t_l_city" : "Moscow",
         "a_phone" : "+7.953535385",
         "a_patronimic_ru" : "Иванович",
         "a_first_name" : "Ivan",
         "t_l_city_ru" : "Москва",
         "t_state_ru" : "Москва",
         "a_city" : "Moscow",
         "o_company" : "Luntik, LLC",
         "o_city_ru" : "Москва",
         "o_l_addr" : "Bolshoy sobachiy, 33",
         "o_l_postcode" : "191000",
         "o_last_name" : "Ivanoww",
         "a_country_code" : "RU",
         "a_l_state_ru" : "Москва",
         "a_patronimic" : "Ivanovich",
         "t_state" : "Moscow",
         "t_last_name" : "Ivanoww",
         "o_country_code" : "RU",
         "t_l_addr_ru" : "Большой собачий переулок, 33",
         "a_email" : "srs-devel@reg.ru",
         "o_city" : "Moscow",
         "a_code" : "6663058366",
         "t_l_state" : "Moscow",
         "o_state_ru" : "Москва",
         "t_addr" : "Bolshoy sobachiy, 33",
         "t_company" : "Luntik, LLC",
         "a_state_ru" : "Москва",
         "t_l_addr" : "Bolshoy sobachiy, 33",
         "t_city_ru" : "Москва",
         "t_postcode" : "191000",
         "a_l_addr_ru" : "Большой собачий переулок, 33",
         "t_l_state_ru" : "Москва",
         "o_l_addr_ru" : "Большой собачий переулок, 33",
         "a_last_name" : "Ivanoww",
         "a_l_postcode" : "191000",
         "o_postcode" : "191000"
      },
      "nss" : {
         "ns1" : "ns2.reg.ru",
         "ns0" : "ns1.reg.ru"
      },
      "domains" : [
         {
            "dname" : "белыйторшер.москва"
         },
         {
            "dname" : "whitelamp.moscow"
         }
      ]
   }
}
```

###### Пример запроса на регистрацию доменов в зоне .САЙТ и .ОНЛАЙН:

```
{
   "input_format" : "json",
   "username" : "test",
   "password" : "test",
   "input_data" : {
      "io_encoding" : "utf8",
      "output_content_type" : "plain",
      "lang" : "en",
      "show_input_params" : 1,
      "domains" : [
         {
            "dname" : "белыйторшер.онлайн"
         },
         {
            "dname" : "белыйторшер.сайт"
         }
      ],
      "contacts" : {
         "b_fax" : "+7.953535385",
         "a_addr_r" : "Большой собачий переулок, 33",
         "t_addr" : "Bolshoy sobachiy, 33",
         "t_last_name_r" : "Иванов",
         "a_state" : "Moscow",
         "o_fax" : "+7.953535385",
         "a_company_r" : "ООО \"Лунтик\"",
         "o_company_r" : "ООО \"Лунтик\"",
         "o_postcode" : "91000",
         "t_phone" : "+7.953535385",
         "a_fax" : "+7.953535385",
         "o_addr" : "Bolshoy sobachiy, 33",
         "b_last_name_r" : "Иванов",
         "o_first_name_r" : "Иван",
         "a_email" : "srs-devel@reg.ru",
         "a_addr" : "Bolshoy sobachiy, 33",
         "t_company" : "Luntik, LLC",
         "b_phone" : "+7.953535385",
         "o_company" : "Luntik, LLC",
         "o_city" : "Moscow",
         "b_first_name" : "Ivan",
         "b_addr_r" : "Большой собачий переулок, 33",
         "t_city" : "Moscow",
         "b_state" : "Moscow",
         "o_last_name_r" : "Иванов",
         "a_postcode" : "91000",
         "t_first_name" : "Ivan",
         "t_state" : "Moscow",
         "b_state_r" : "Москва",
         "t_email" : "srs-devel@reg.ru",
         "a_country_code" : "RU",
         "o_phone" : "+7.953535385",
         "t_company_r" : "ООО \"Лунтик\"",
         "a_city" : "Moscow",
         "t_country_code" : "RU",
         "b_last_name" : "Ivanoww",
         "a_phone" : "+7.953535385",
         "t_postcode" : "91000",
         "b_first_name_r" : "Иван",
         "a_last_name_r" : "Иванов",
         "t_state_r" : "Москва",
         "t_last_name" : "Ivanoww",
         "a_city_r" : "Москва",
         "a_state_r" : "Москва",
         "b_addr" : "Bolshoy sobachiy, 33",
         "a_first_name_r" : "Иван",
         "b_company" : "Luntik, LLC",
         "o_city_r" : "Москва",
         "o_last_name" : "Ivanoww",
         "b_country_code" : "RU",
         "a_company" : "Luntik, LLC",
         "b_email" : "srs-devel@reg.ru",
         "a_last_name" : "Ivanoww",
         "t_addr_r" : "Большой собачий переулок, 33",
         "b_city" : "Moscow",
         "o_state" : "Moscow",
         "o_first_name" : "Ivan",
         "b_company_r" : "ООО \"Лунтик\"",
         "t_city_r" : "Москва",
         "t_fax" : "+7.953535385",
         "o_state_r" : "Москва",
         "o_addr_r" : "Большой собачий переулок, 33",
         "b_postcode" : "91000",
         "o_country_code" : "RU",
         "o_email" : "srs-devel@reg.ru",
         "a_first_name" : "Ivan",
         "b_city_r" : "Москва",
         "t_first_name_r" : "Иван"
      },
      "nss" : {
         "ns0" : "ns1.reg.ru",
         "ns1" : "ns2.reg.ru"
      },
      "output_format" : "json"
   }
}
```

##### Примеры успешных ответов:

пример ответа на первый запрос (запрос в «PLAIN» формате):

```
{
   "answer" : {
      "bill_id" : "1234",
      "domains" : [
         {
            "dname" : "vschizh.ru",
            "result" : "success",
            "service_id" : "12345"
         }
      ],
      "pay_notes" : "Amount successfully charged",
      "pay_type" : "prepay",
      "payment" : "600"
   },
   "result" : "success"
}
```

пример ответа на второй запрос (запрос в JSON формате):

```
{
   "answer" : {
      "bill_id" : "1234",
      "domains" : [
         {
            "dname" : "vschizh.su",
            "result" : "success",
            "service_id" : "12345"
         }
      ],
      "pay_notes" : "Amount successfully charged",
      "pay_type" : "prepay",
      "payment" : "600"
   },
   "result" : "success"
}
```

пример ответа на третий запрос (оптовый запрос в JSON формате):

```
{
   "answer" : {
      "bill_id" : "1234",
      "domains" : [
         {
            "dname" : "vschizh.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "vschizh.su",
            "result" : "success",
            "service_id" : "12346"
         }
      ],
      "pay_notes" : "Amount successfully charged",
      "pay_type" : "prepay",
      "payment" : "1300"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [стандартные коды ошибок](#common_error_codes)

## 8.8. Функция: domain/transfer

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

подать заявку на перенос домена от другого регистратора

##### Поддержка обработки списка услуг

Только для VIP клиентов.

##### [Поля запроса:](#common_input_params)

Совпадают с полями для функции [domain/create](#domain_create). Помимо этого для доменов .RU / .SU / .РФ зон данные владельца домена и DNS-сервера можно не указывать, т.е. оставлять хеши contacts и nss пустыми. Эти данные будут автоматически получены из реестра в момент принятия домена. Также для доменов .RU / .SU / .РФ поле period может принимать значение "0" (перенос доменов без продления). Для большинства международных доменов (com net org info biz mobi name asia tel in mn bz cc tv us me cn nz co ca и др.) дополнительно надо указать authinfo.

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| authinfo | 5 | 48 | Ключ аутентификации для переноса домена. Уточняется у предыдущего регистратора домена. Для .RU/.РФ зон минимальная длинна 6 знаков |

###### Контактные данные для .RU/.РФ доменов

Будут использоваться для сверки с данными, внесёнными в реестр текущим регистратором

###### Данные частного лица (только при регистрации домена на частное лицо!)

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| person\_r\_surname | 1 | 32 | Фамилия администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит фамилию в оригинальном написании (при невозможности в английской транслитерации).  *Пример1: Пупкин*  *Пример2: Smith* |
| person\_r\_name | 1 | 32 | Имя администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит имя в оригинальном написании (при невозможности в английской транслитерации).  *Пример1: Василий*  *Пример2: John* |
| person\_r\_patronimic | 1 | 32 | Отчество администратора домена на русском языке в соответствии с паспортными данными. Для иностранцев поле содержит отчество в оригинальном написании (при невозможности в английской транслитерации).  *Пример1: Николаевич*  *Пример2: R* |
| birth\_date | 10 | 10 | Дата рождения администратора домена в формате ДД.ММ.ГГГГ.  *Пример: 07.11.1917* |
| passport\_number | 6 | 30 | Серия и номер паспорта. В написании римских цифр допустимо использование только латинских букв. Знак номера перед номером паспорта не ставится. Паспорта СССР (паспорта старого образца) не принимаются.  *Пример: 34 02 651241* |
| passport\_place | 3 | 200  много-  строчное | Наименование органа выдавшего паспорт. Запись может быть многострочной.  *Пример: 48 о/м г.Москвы* |
| passport\_date | 10 | 10 | Дата выдачи паспорта. Дата записывается в формате ДД.ММ.ГГГГ.  *Пример: 26.12.1992* |
| p\_addr\_zip | 2 | 15 | Почтовый индекс администратора домена.  *Пример: 101000* |
| p\_addr\_area | 3 | 65 | Область из почтового адреса администратора домена.  *Пример: Московская обл.* |
| p\_addr\_city | 3 | 25 | Название города из почтового адреса администратора домена.  *Пример: г. Москва* |
| p\_addr\_addr | 5 | 105 | Адрес включающий название улицы и номер дома (возможна доп. информация в виде номера корпуса/квартиры и т.д.).    *Пример: ул.Пупкина, 1, стр. 2, отдел мебели, офис 433* |
| p\_addr\_recipient | 5 | 45 | Имя получателя(физ лицо либо, название отдела)  *Пример: В. Лоханкин* |
| country | 2 | 2 | Двухбуквенный ISO-код страны, гражданином которой является частное лицо.  *Пример: RU* |

###### Данные организации (только при регистрации домена на организацию!)

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| org\_r | 10 | 255  много-  строчное | Полное наименование организации-администратора домена на русском языке в соответствии с учредительными документами. Для нерезидентов указывается написание на национальном языке (либо на английском языке). Запись может быть многострочной.  *Пример1: Урюпинский государственный университет\nимени Карлы-Марлы*  *Пример2: Общество с ограниченной ответственностью "Рога и Копыта"* |
| code | 10 | 10 | Идентификационный номер налогоплательщика (ИНН), присвоенный организации-администратору. Запись может содержать пустую строку, если администратором является нерезидент РФ, не имеющий идентификационного номера налогоплательщика.  *Пример: 7701107259* |
| country | 2 | 2 | Двухбуквенный ISO-код страны, в которой зарегистрирована организация.  *Пример: RU* |
| address\_r\_zip | 2 | 15 | Почтовый индекс юридического адреса организации в соответствии с учредительными документами.  *Пример: 101000* |
| address\_r\_area | 3 | 65 | Область из юридического адреса организации в соответствии с учредительными документами.  *Пример: Московская обл.* |
| address\_r\_city | 3 | 25 | Название города юридического адреса организации в соответствии с учредительными документами.  *Пример: г. Москва* |
| address\_r\_addr | 5 | 105 | Адрес включающий название улицы и номер дома (возможна доп. информация в виде номера корпуса/квартиры и т.д.) юридического адреса организации в соответствии с учредительными документами.  *Пример: ул.Пупкина, 1, стр. 2* |

##### [Поля ответа:](#common_response_parameters)

Совпадают с полями для функции [domain/create](#domain_create)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| AUTHINFO\_NOT\_FOUND | Transfer secretkey not found | Секретный ключ для переноса не найден |
| INVALID\_AUTHINFO | Inadmissible chars in authinfo | Недопустимые символы в поле authinfo |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.9. Функция: domain/get\_transfer\_status

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

проверка состояния переноса домена, работает только для тех зон, где REG.RU является регистратором, статус в остальных зонах надо проверять через whois

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Массив со списком хешей, содержащий имена доменов dname и их статус. |
| status | Cтатус переноса, возможные статусы: pendingPayment — счёт ожидает оплаты, перенос ещё не начинался; pendingConfirmation — требуется подтверждение переноса; authinfoError — неверный ключ переноса, требуется ввести правильный ключ; clientTransferProhibited — текущий регистратор заблокировал перенос домена; transferProhibited — домен имеет статус, блокирующий перенос; transferError — какая-то другая ошибка переноса; pendingTransfer — домен в процессе переноса; alreadyTransfered — домен уже был перенесён; notAvailable — определение статуса не поддерживается. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_transfer_status
input_data={"domains":[{"dname":"ya.ru"},{"dname":"qwerty.ooo"},{"dname":"xn--41a.com"},{"dname":"zzz.рф"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "ya.ru",
            "error_code" : "DOMAIN_NOT_FOUND",
            "error_params" : {
               "domain_name" : "ya.ru"
            },
            "error_text" : "Domain ya.ru not found or not owned by You",
            "result" : "error"
         },
         {
            "dname" : "qwerty.ooo",
            "result" : "success",
            "service_id" : "12345",
            "status" : "notAvailable"
         },
         {
            "dname" : "я.com",
            "result" : "success",
            "service_id" : "12346",
            "status" : "pendingTransfer"
         },
         {
            "dname" : "zzz.рф",
            "error_code" : "INVALID_DOMAIN_NAME_FORMAT",
            "error_params" : {
               "domain_name" : "zzz.xn--p1ai"
            },
            "error_text" : "domain_name is invalid or unsupported zone",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.10. Функция: domain/set\_new\_authinfo

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

замена неверного ключа переноса (authinfo) домена, доступно только до отправки запроса в реестр

##### [Поля запроса:](#common_input_params)

| Параметр | Минимальная длина поля | Максимальная длина поля | Описание |
| --- | --- | --- | --- |
| authinfo | 5 | 32 | ключ переноса домена, для .RU/.РФ зон минимальная длинна 6 знаков |

А также [стандартные параметры идентификации услуги](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

нет

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/set_new_authinfo
authinfo=hjgf7jfj8f8
dname=test.com
output_content_type=plain
output_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "dname" : "test.com",
      "service_id" : "12345"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| AUTHINFO\_NOT\_FOUND | Transfer secretkey not found | Секретный ключ для переноса не найден |
| INVALID\_AUTHINFO | Inadmissible chars in authinfo | Недопустимые символы в поле authinfo |
| WAITING | Service already ordered but not processed. | Услуга уже заказана, но еще не отработана. |
| NOT\_A\_TRANSFER\_ORDER | Domain was created on your account without transfer | Домен был создан на вашем аккаунте без переноса |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.11. Функция: domain/get\_rereg\_data

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получить список освобождающихся доменов с характеристиками, данные обновляются 1 раз в 30 минут

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domains | Список доменов |
| dname\_matching | Шаблон поиска по именам доменов |
| search\_query | Поисковые запросы |
| max\_chars | Максимально количество символов в имени домена |
| min\_pr | Минимальное значение Google PR |
| min\_cy | Минимальное значение Яндекс тИЦ |
| kley | 1 - показывать только те домены, где зеркала не обнаружены |
| zone | Доменная зона (ru, su, рф) |
| views | Количество просмотров в месяц |
| vis | Количество уникальных посетителей в месяц |
| traf | Поисковый трафик |
| price | Максимальная текущая цена |
| registrar | Регистратор |
| delete\_date | Дата удаления |
| sortcol | Сортировка по: name - по имени, price - по блиц-цене, fdate - по дате создания, date - по дате удаления, pr - по Google PR, cy - по Яндекс тИЦ, views - по количеству просмотров в месяц, vis - по числу уникальных посетителей, sch - по среднему количеству поискового трафика за весь период, reg - по регистратору |
| premium | 1 - показать только премиум домены |
| limit | Сколько отдавать записей за 1 запрос, значение по умолчанию — 1000, максимальное значение — 1000 |
| limit\_from | С какой позиции (записи) отдавать данные, значение по умолчанию — 0 |

Полный список всех освобождающихся доменов одним файлом в CSV формате можно скачать [тут](https://www.reg.ru/static_files/rereg_extlist.csv)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| dname | Имя домена |
| tld | Доменная зона |
| first\_create\_date | Первичная регистрация |
| delete\_date | Дата удаления |
| start\_price | Минимальная ставка (для крупных клиентов) |
| bid | Последняя ставка (для крупных клиентов) |
| price | Максимальная ставка |
| uni\_avg\_attendance | Среднее кол-во посетителей за день |
| avg\_viewings | Среднее кол-во просмотров за день |
| all\_avg\_traffic | Средний трафик за день |
| search\_query\_list | Список поисковых запросов |
| yandex\_tic | Яндекс тИЦ |
| tic\_mirrors | Зеркала у Яндекс тИЦ |
| google\_pr | Google Page Rank |
| registrar | Регистратор |
| is\_recommended | Премиум домен |

* Дополнительные поля ответа доступны для пользователей, у которых 256 и более активных доменов на аккаунте. Проверка количества доменов производится раз в месяц в автоматическом режиме.

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_rereg_data
input_data={"domains":[{"dname":"qqq.ru"},{"dname":"eee.ru"}],"min_pr":"2","sortcol":"vis"}
input_format=json
output_content_type=plain
output_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : [
      {
         "dname" : "test.ru",
         "tld" : "ru",
         "yandex_tic" : "1000",
         "tic_mirrors" : "null",
         "uni_avg_attendance" : "100",
         "search_query_list" : "слово дело",
         "start_price" : "2500.00",
         "bid" : "2500.00",
         "avg_viewings" : "2",
         "google_pr" : "9",
         "price" : "2500.00",
         "blitz_price" : "2500.00",
         "first_create_date" : "2001-01-01",
         "delete_date" : "2020-01-01",
         "is_recommended" : "1",
         "all_avg_traffic" : "3"
         "registrar" : "REGRU-RU"
      },
      {
         "dname" : "test.su",
         "tld" : "su",
         "yandex_tic" : "0",
         "tic_mirrors" : "www.test.su",
         "uni_avg_attendance" : "0",
         "search_query_list" : null,
         "start_price" : "600.00",
         "bid" : "600.00",
         "avg_viewings" : "0",
         "google_pr" : "0",
         "price" : "600.00",
         "blitz_price" : "2500.00",
         "first_create_date" : "1991-08-21",
         "delete_date" : "2020-01-01",
         "is_recommended" : "0",
         "all_avg_traffic" : "0"
         "registrar" : "REGRU-RU"
      }
   ],
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.12. Функция: domain/set\_rereg\_bids

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Сделать ставки на освобождающиеся домены.

Подробнее смотрите [здесь](/domain/new/rereg)

##### Доступность:

клиенты

Так же партнёры компании REG.RU могут внести предварительную оплату в размере 225 руб., а остаток суммы оплатить в случае успешного выполнения заявки на регистрацию в течение 10 дней после её исполнения (см. ниже параметр instalment). При выборе данного порядка оплаты домен будет зарегистрирован на REG.RU, а после оплаты полной стоимости автоматически перерегистрирован на указанные Вами данные. В случае невыполнения обязательств по оплате полной стоимости в установленный срок предварительная оплата будет считаться неустойкой и не подлежит возмещению. Подробнее см. Договор пп. 2.16, 2.17, 3.2.9, 6.11.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| contacts | Хеш контактных данных; описание контактных данных, являющихся ключами хеша, см. в функции [domain/create](#domain_create). |
| nss | Хеш NS-серверов; описание формата NS-серверов, являющихся ключами хеша, см. в функции [domain/create](#domain_create). |
| domains | Массив со списком доменов, каждый элемент массива является хешем с ключами dname — имя домена и price — ценой/ставкой на этот домен. Для партнёров доступен заказ по частичной оплате, ключ instalment, допустимые значения 0 и 1, в этом случае можно установить автоматическую оплату после активации домена при наличии денег на счёте, ключ autopay, допустимые значения 0 и 1. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| bill\_id | Номер счёта, созданного по запросу. |
| payment | Сумма заказа в рублях. |
| pay\_type | Способ оплаты. На данный момент возможна только предоплата, prepay. |
| pay\_notes | Комментарий, относящийся к используемому способу оплаты. |
| domains | Список доменов с результатом по каждому, поле результатов может иметь такие значения:  success — ставка сделана,  Invalid domain zone — для этой зоны нет возможности заказа освобождающихся доменов,  Domain not found — домен не обслуживается REG.RU,  Rereg not found — домен не найден в списке доступных освобождающихся доменов (например, он уже имеет предельную ставку),  Invalid bid — недопустимая ставка,  More bid found — найдена такая же или большая ставка. |

##### Пример запроса:

пример отправляемых данных (структура на Perl-e) с преобразованием её в JSON-формат:

```
$jsondata = {
    contacts => {
        descr      => 'Vschizh site',
        person     => 'Svyatoslav V Ryurik',
        person_r   => 'Рюрик Святослав Владимирович',
        passport   => '22 44 668800, выдан по месту правления 01.09.1164',
        birth_date => '01.01.1101',
        p_addr     => '12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому',
        phone      => '+7 495 1234567',
        e_mail     => 'test@test.ru',
        country    => 'RU',
    },
    nss => {
        ns0 => 'ns1.reg.ru',
        ns1 => 'ns2.reg.ru',
    },
    domains => [
        { dname => 'vschizh.ru', price => 225 },
        # или заказ в рассрочку: { dname => 'vschizh.ru', price => 2500, instalment => 1 },
        { dname => 'vschizh.su', price => 400 },
    ],
};
$jsondata = JSON::XS->new->utf8->encode( $jsondata );
```

Сам запрос:

```
https://api.reg.ru/api/regru2/domain/set_rereg_bids
input_data={"contacts":{"country":"RU","e_mail":"test@test.ru","person_r":"Рюрик Святослав Владимирович","phone":"+7 495 1234567","birth_date":"01.01.1101","descr":"Vschizh site","person":"Svyatoslav V Ryurik","p_addr":"12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому","passport":"22 44 668800, выдан по месту правления 01.01.1164"},"domains":[{"dname":"vschizh.ru","price":225},{"dname":"vschizh.su","price":400}],"nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"}}
input_format=json
output_content_type=plain
password=test
username=test
```

То же самое, но с использованием wget для post-запроса (параметр --post-data) и выводом ответа в консоль:

```
wget -O - https://api.reg.ru/api/regru2/domain/set_rereg_bids \
--post-data='username=test&password=test&input_format=json&\
input_data={"contacts":{"country":"RU","e_mail":"test@test.ru", \
"person_r":"Рюрик Святослав Владимирович","phone":"%2B7 495 1234567","birth_date":"01.01.1101", \
"descr":"Vschizh site","person":"Svyatoslav V Ryurik",\
"p_addr":"12345, г. Вщиж, ул. Княжеска, д.1, Рюрику Святославу Владимировичу, князю Вщижскому",\
"passport":"22 44 668800, выдан по месту правления 01.01.1164"},\
"domains":[{"dname":"vschizh.ru","price":225},{"dname":"vschizh.su","price":400}],\
"nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"}}'
```

Обратите внимание, что если Ваша библиотека/программа не делает полного автоматического преобразования данных, то «+» надо передавать как «%2B».

##### Пример ответа:

результат для приведенного выше запроса wget:

```
{
   "answer" : {
      "bill_id" : "1234",
      "domains" : [
         {
            "dname" : "vschizh.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "vschizh.su",
            "result" : "success",
            "service_id" : "12346"
         }
      ],
      "pay_notes" : "Amount successfully charged",
      "pay_type" : "prepay",
      "payment" : "625"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| CONTACTS\_NOT\_FOUND | Contacts list not found | Контактная информация не найдена |
| INVALID\_CONTACTS | Contacts user data is invalid | Ошибка в контактных данных пользователя |
| UNKNOWN\_CONTYPE | Can't guess registrant type: person or organization | Не определён тип контакта: физ.лицо или организация |
| DOMAINS\_NOT\_FOUND | Domains list not found | Список доменов не найден |
| REREG\_NOT\_FOUND | Domain $domain\_name in expiring list not found | Домен $domain\_name в списке освобождающихся не найден |
| INVALID\_BID | Invalid bid | Неверная ставка |
| SAME\_OR\_MORE\_BID\_FOUND | The same or big rate is found | Найдена такая же или большая ставка |
| YOUR\_SAME\_OR\_MORE\_BID\_FOUND | Your same or big rate is found | Найдена ваша такая же или большая ставка |

А также см. [cтандартные коды ошибок](#common_errors)

## 8.13. Функция: domain/get\_user\_rereg\_bids

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получить список освобождающихся доменов со своими ставками, в списке так же присутсвуют домены по которым ставка перебита

##### [Поля запроса:](#common_input_params)

нет

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| dname | Имя домена |
| tld | Доменная зона |
| is\_active | Возможны ли торги по данному лоту |
| is\_recommended | Премиум домен |
| uni\_avg\_attendance | Среднее кол-во посетителей за день |
| avg\_viewings | Среднее кол-во просмотров за день |
| all\_avg\_traffic | Средний трафик за день |
| search\_query\_list | Список поисковых запросов |
| first\_creation\_date | дата самой первой регистрации домена по данным statonline.ru |
| delete\_date | Дата удаления из реестра |
| yandex\_tic | Яндекс тИЦ |
| google\_pr | Google Page Rank |
| registrar | Регистратор, у которого домен находится сейчас на обслуживании |
| user\_bid | Ваша последняя ставка |
| max\_bid | Максимальная ставка на этот домен, если лот ещё активен, то вы можете её перебить независимо от того кто её сделал |
| next\_price | Следующая минимальная ставка, имеет значение NULL для неактивных лотов |
| blitz\_price | Максимальная ставка, при её достижении домен снимается с торгов (is\_active = 0), но будет присутствовать в этом списке до момента освобождения домена реестром |
| rereg\_bids | список всех возможных ставок отдельно по каждой доменной зоне (для справки) |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_user_rereg_bids
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
    answer => {
        domains => [
            {
                delete_date        => '2099-12-31',
                yandex_tic         => null,
                max_bid            => 225.00,
                next_price         => 590.00,
                uni_avg_attendance => null,
                blitz_price        => 8000.00,
                user_bid           => 225.00,
                search_query_list  => null,
                tld                => 'ru',
                dname              => 'test.ru',
                registrar          => 'REGRU',
                is_active          => 1,
                avg_viewings       => null,
                first_create_date  => '2000-01-01',
                google_pr          => null,
                is_recommended     => 1,
                all_avg_traffic    => null
            },
            {
                delete_date        => '2099-01-01',
                yandex_tic         => 2000,
                max_bid            => 5000.00,
                next_price         => null,
                uni_avg_attendance => null,
                blitz_price        => 5000.00,
                user_bid           => 5000.00,
                search_query_list  => null,
                tld                => 'su',
                dname              => 'test.su',
                registrar          => 'REGRU',
                is_active          => 0,
                avg_viewings       => null,
                first_create_date  => '2001-01-01',
                google_pr          => 9,
                is_recommended     => 1,
                all_avg_traffic    => null
            }
        ],
        rereg_bids => {
            рф => [ 0, 225, 590, 750, 2500, 5000, 8000 ],
            su => [ 0, 400, 590, 750, 2500, 5000, 8000 ],
            ru => [ 0, 225, 590, 750, 2500, 5000, 8000 ]
        }
    },
    result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.14. Функция: domain/get\_docs\_upload\_uri

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получение ссылки на закачивание документов из интернета для .RU/.SU/.РФ доменов

##### [Поля запроса:](#common_input_params)

[стандартные поля для идентификации домена](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| docs\_upload\_sid | идентификатор закачиваемого документа |
| url | ссылка для закачивания документа, включает в себя идентификатор docs\_upload\_sid |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_docs_upload_uri
dname=test.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "docs_upload_sid" : "123456",
      "url" : "http://www.reg.ru/user/docs/add?userdoc_secretkey=123456"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| CANT\_GET\_DOCS\_UPLOAD\_SID | Can't get documents upload sid. | Не удалось получить sid загрузки документов. |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.15. Функция: domain/update\_contacts

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Изменение контактных данных домена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| contacts | Хеш контактных данных; описание контактных данных, являющихся ключами хеша, см. в функции [domain/create](#domain_create). Нужно только в случае запроса в форматах JSON, XML |
| domains | Массив со списком доменов, каждый элемент массива является хешем с ключём dname, имя домена, или service\_id. Домены в списке должны быть однотипные, т.е. все относится к одной доменной зоне или принадлежать к одной из групп:  – .ru, .su и тип контактных данных person;  – .ru, .su и тип контактных данных org;  – .com.ua, .kiev.ua;  – .com, .net, .org, .biz, .info, .name, .mobi.  Нужно только в случае запроса в форматах JSON, XML |
| payout\_agreement | Флаг согласия клиента на списание денег с его счета. Только для доменных зон с платной сменой контактных данных. В настоящее время, это зона MD. Для MD стоимость смены контактных данных составляет 4800 рублей. Для EU требуется продление регистрации домена на 1 год. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов с параметрами dname, service\_id и/или error\_code c кодом ошибки |

##### Пример запроса:

Запрос с одним доменом

```
https://api.reg.ru/api/regru2/domain/update_contacts
birth_date=01.01.2000
country=RU
descr=Vschizh site
domain_name=vschizh.ru
e_mail=test@test.ru
output_content_type=plain
p_addr_addr=ул. Княжеска, д.1
p_addr_area=
p_addr_city=г. Вщиж
p_addr_recipient=Рюрику Святославу Владимировичу
p_addr_zip=123456
passport_date=01.01.1164
passport_number=22 44 668800
passport_place=выдан по месту правления
password=test
person=Svyatoslav V Ryurik
person_r_name=Святослав
person_r_patronimic=Владимирович
person_r_surname=Рюрик
phone=%2B7 495 1234567
username=test
```

Запрос со списком доменов

```
https://api.reg.ru/api/regru2/domain/update_contacts
input_data={"username":"test","password":"test","contacts":{"country":"RU","e_mail":"test@test.ru","person_r_surname":"Рюрик","person_r_name":"Святослав","person_r_patronimic":"Владимирович","phone":"%2B7 495 1234567","birth_date":"01.01.2000","descr":"Vschizh site","person":"Svyatoslav V Ryurik","p_addr_zip":"123456","p_addr_area":"","p_addr_city":"г. Вщиж","p_addr_addr":"ул. Княжеска, д.1","p_addr_recipient":"Рюрику Святославу Владимировичу","passport_number":"22 44 668800","passport_place":"выдан по месту правления","passport_date":"01.01.2000"},"domains":[{"dname":"vschizh.ru"},{"dname":"vschizh.su"}]}
input_format=json
output_content_type=plain
```

##### Пример ответа:

```
{
    answer => {
        domains => [
            {
                dname      => 'vschizh.ru',
                service_id => '12345',
                result     => 'success'
            },
            {
                dname      => 'vschizh.su',
                service_id => '12346',
                result     => 'success'
            }
        ],
   },
   "charset" : "utf-8",
   "messagestore" : null,
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INCOMPATIBLE\_CONTYPES | Incompatible .ru/.su/.рф domain contypes | Несовместимые типы контактов для .ru/.su/.рф доменов |
| INCOMPATIBLE\_ZONES | Incompatible domain zones | Несовместимые доменные зоны |
| PP\_UPDATE\_FAIL | Update Private Person is fail: $error\_detail | Изменить флаг Private Person не удалось: $error\_detail |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.16. Функция: domain/update\_private\_person\_flag

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Изменение флага Private Person скрытия/отображения контактных данных в whois

##### [Поля запроса:](#common_input_params)

| Параметр | Значения | Описание |
| --- | --- | --- |
| private\_person\_flag | 0 и 1 | Установка/снятие флага скрытия персональных данных |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов с параметрами dname, service\_id и/или error\_code c кодом ошибки |
| pp\_flag | соответствует переданному значению входного параметра private\_person\_flag,  возможные ответы: 'is set' и 'is cleared' |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/update_private_person_flag
input_data={"username":"test","password":"test","domains":[{"dname":"vschizh.ru"},{"dname":"vschizh.su"}],"private_person_flag":"1","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
  answer => {
      domains => [
          {
              dname      => 'vschizh.ru',
              service_id => '12345',
              result     => 'success'
          },
          {
              dname      => 'vschizh.su',
              service_id => '12346',
              result     => 'success'
          }
      ],
      pp_flag => 'is set'
  }
  result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.17. Функция: domain/register\_ns

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

внесение домена в NSI-registry, работает только для международных доменов

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domain\_name | домен, nameserver которого будет добавляться |
| ns0 | nameserver |
| ns0ip | IP адрес добавляемого nameserver-а |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| resp | детализированный ответ регистратора NSI-доменов, присутствует при положительном ответе, обычно является хешем |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/register_ns
dname=test.com
ns0=ns0.test.com
ns0ip=1.2.3.4
output_content_type=plain
output_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "resp" : {
         "actionstatus" : "success",
         "actionstatusdesc" : "Addition Completed Successfully",
         "actiontype" : "AddCns",
         "actiontypedesc" : "Addition of Child Nameserver ns0.test.com with IP [1.2.3.4]",
         "description" : "test.com",
         "status" : "success"
      }
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.18. Функция: domain/delete\_ns

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

удаление домена из NSI-registry, поддерживаются только для международных доменов

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domain\_name | домен, nameserver которого будет удаляться |
| ns0 | nameserver |
| ns0ip | IP адрес удаляемого nameserver-а |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| resp | детализированный ответ регистратора NSI-доменов, присутствует при положительном ответе, обычно является хешем |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/delete_ns
dname=test.com
ns0=ns0.test.com
ns0ip=1.2.3.4
output_content_type=plain
output_format=json
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "resp" : {
         "actionstatus" : "success",
         "actionstatusdesc" : "Modification Completed Successfully",
         "actiontype" : "DelCnsIp",
         "actiontypedesc" : "Deletion of IP Address [1.2.3.4] from Child Nameserver ns0.test.com",
         "description" : "test.com",
         "status" : "success"
      }
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.19. Функция: domain/get\_nss

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Получение DNS для доменов

##### [Поля запроса:](#common_input_params)

[Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Список доменов с параметрами dname, service\_id и nss, или error\_code c кодом ошибки идентификации услуги. |
| nss | Список DNS. Для ns на базе самого доменного имени, дополнительно возращается его ip адрес. Если у такого ns более одного ip, то адреса возвращаются массивом. |

##### Пример запроса:

Один домен

```
https://api.reg.ru/api/regru2/domain/get_nss
domain_name=test.ru
output_content_type=plain
password=test
username=test
```

Несколько доменов

```
https://api.reg.ru/api/regru2/domain/get_nss
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.su"}],"output_content_type":"plain"}
input_format=json
```

##### Примеры успешного ответа:

```
{
    answer => {
        domains => [
            {
                dname      => 'test.ru',
                nss        => [
                    {
                        ns => 'ns1.reg.ru'
                    },
                    {
                        ns => 'ns2.reg.ru'
                    }
                ],
                service_id => 12345
            }
        ]
    },
    result => 'success'
}
```

```
{
    answer => {
        domains => [
            {
                dname      => 'test.ru',
                nss        => [
                    {
                        ns => 'ns1.reg.ru'
                    },
                    {
                        ns => 'ns2.reg.ru'
                    }
                ],
                service_id => 12345
            },
            {
                dname      => 'test.su',
                nss        => [
                    {
                        ns => 'ns1.reg.ru'
                    },
                    {
                        ns => 'ns2.reg.ru'
                    }
                ],
                service_id => 12346
            }
        ]
    },
    result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.20. Функция: domain/update\_nss

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Изменение DNS серверов домена, установка/снятие делегирования домена (только для партнёров)

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domain\_name | Имя домена |
| ns0...ns3 | Имена DNS серверов в порядке убывания приоритета. |
| ns0ip...ns3ip | IP адреса DNS серверов. Необязательные поля. Используются только если имя DNS сервера содержит имя регистрируемого домена.  В некоторых зонах разрешено указывать более одного IP адреса для DNS сервера. Список IP адресов можно передавать в виде одной строки с разделителем «,», либо в виде массива. |
| nss | Хеш, содержащий список NS-серверов и, если надо, IP адресов. В ключами хеша будут поля ns0...ns3 и ns0ip...ns3ip, описанные выше. Только для запросов в JSON/XML формате |
| undelegate | Установка/снятие домена с делегирования.  0 - Делегировать,  1 - Снять делегирование |
| payout\_agreement | Флаг согласия клиента на списание денег с его счета. Только для доменных зон с платной сменой DNS-серверов. В настоящее время, это зона MD. Стоимость смены контактных данных для этой зоны составляет 4800 рублей. |

Также см. [параметры идентификации списка услуг](#common_service_identification_params).

Примечание:

Для поддержки DNS могут быть бесплатно использованы сервера REG.RU. Для этого в качестве DNS-серверов необходимо указать сервера `ns1.reg.ru` и `ns2.reg.ru`. В этом случае на серверах REG.RU будет прописана зона для Вашего домена. Управлять зоной можно через web-интерфейс сайта REG.RU или через API.

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов с параметрами dname и service\_id, или error\_code c кодом ошибки идентификации услуги |

##### Примеры запросов:

Изменение DNS одного домена

PLAIN:

```
https://api.reg.ru/api/regru2/domain/update_nss
dname=test.ru
ns0=ns1.test.com
ns0ip=1.2.3.4
ns1=ns2.test.com
ns1ip=2.3.4.5,3.4.5.6
output_content_type=plain
password=test
username=test
```

JSON:

```
https://api.reg.ru/api/regru2/domain/update_nss
input_data={"dname":"test.ru","nss":{"ns0":"ns1.test.ru", "ns1":"ns2.test.ru", "ns0ip":"1.2.3.4,2.3.4.5", "ns1ip":["3.4.5.6","4.5.6.7"]}}
input_format=json
password=test
username=test
```

Изменение DNS у списка доменов, один из которых содержит ошибку в названии (ответ для такого случая см. ниже)

```
https://api.reg.ru/api/regru2/domain/update_nss
input_data={"domains":[{"dname":"test.ru"},{"dname":"test.su"},{"dname":"----.ru"}],"nss":{"ns0":"ns1.reg.ru","ns1":"ns2.reg.ru"},"output_content_type":"plain"}
input_format=json
password=test
username=test
```

Снятие домена с делегирования без передачи NS-серверов

```
https://api.reg.ru/api/regru2/domain/update_nss
domain_name=test.ru
output_content_type=plain
password=test
undelegate=1
username=test
```

##### Пример ответа:

Один домен

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         }
      ]
   },
   "result" : "success"
}
```

Несколько доменов, один из которых содержит ошибочные данные:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.su",
            "result" : "success",
            "service_id" : "12346"
         },
         {
            "dname" : "----.ru",
            "error_code" : "INVALID_DOMAIN_NAME_FORMAT",
            "result" : "domain_name is invalid or unsupported zone"
         }
      ]
   },
   "result" : "success"
}
```

Успешный ответ смены флага делегирования домена без NS-серверов (запрос см. выше)

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INVALID\_NSS | Invalid name servers | Неверные NS-сервера |
| UNDELEGATION\_NOT\_SUPPORTED\_BY\_TLD | Delegation/undelegation is not supported in this tld | Зона домена не поддерживает делегирование/разделегирование |
| UNDELEGATION\_NOT\_SUPPORTED\_BY\_DOMAIN\_STATE | Delegation/undelegation locked. Contact technical support for more information. | Делегирование/разделегирование заблокировано. Обратитесь в техническую поддержку за подробной информацией. |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.21. Функция: domain/delegate

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Установка флага делегирования домена

##### [Поля запроса:](#common_input_params)

[Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | список услуг с параметрами dname и service\_id, или error\_code c кодом ошибки идентификации услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/delegate
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.su"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.su",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.22. Функция: domain/undelegate

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Снятие флага делегирования домена

##### [Поля запроса:](#common_input_params)

[Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| services | список услуг с параметрами dname и service\_id, или error\_code c кодом ошибки идентификации услуги |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/undelegate
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.su"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.su",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.23. Функция: domain/transfer\_to\_another\_account

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Полная передача домена на другой аккаунт

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| new\_user\_name | login клиента, на который передаются домены |
| set\_me\_as\_referrer | назначить себя рефералом для передаваемого домена (подробнее — см. [Реферальная программа](/reseller/referral_program)), допустимые значения 0 и 1 |

А также [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов, для которых передаются на другой аккаунт; в случае успешного выполнения запроса на передачу поле result будет содержать request\_is\_sent для каждого домена, иначе — код ошибки |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/transfer_to_another_account
dname=test.su
new_user_name=not_test
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "request_is_sent",
            "service_id" : "12345"
         },
         {
            "dname" : "test.su",
            "error_code" : "NOT_TRANSFERED_DOMAIN_STATUS",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| OPERATION\_FOR\_SERVICE\_OWNER\_ONLY | Only owner can have access to this function | Только владелец имеет доступ к этой функции |
| NOT\_TRANSFERED\_DOMAIN\_STATUS | The status of the domain doesn't admit transfer, probably at first it is necessary to prolong the domain | Статус домена не допускает переноса, возможно сначала надо продлить домен |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.24. Функция: domain/look\_at\_entering\_list

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

просмотр передаваемых на аккаунт доменов

##### [Поля запроса:](#common_input_params)

нет

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| messages | Список сообщений о передаче доменов, каждое сообщение содержит свой идентификатор id и имя передаваемого домена domain\_name, для любой повторной передачи домена идентификатор будет иметь новое значение |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/look_at_entering_list
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "messages" : [
         {
            "domain_names" : [
               "test.ru"
            ],
            "id" : "123456"
         },
         {
            "domain_names" : [
               "test.su",
               "test.com"
            ],
            "id" : "123457"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.25. Функция: domain/accept\_or\_refuse\_entering\_list

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Принять или отклонить передаваемые на этот аккаунт домены

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| id | идентификатор сообщения о передаче домена, получить идентификатор можно при помощи функции [domain/look\_at\_entering\_list](#domain_look_at_entering_list) |
| action\_type | тип действия по данному домену: принять домен accept или отказаться refuse, так же допустимы значения yes и no, 1 и 0 |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Список доменов c результатом действий по каждому домену |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/accept_or_refuse_entering_list
action_type=yes
id=123456
output_content_type=plain
password=test
username=test
```

```
https://api.reg.ru/api/regru2/domain/accept_or_refuse_entering_list
input_data={"domains":[{"id":"123456","action_type":"accept"},{"id":"123457","action_type":"refuse"},{"id":"123458","action_type":"xxx"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

Примеры успешного ответа:

```
{
    answer => {
        domains => [
            {
                id          => 123456,
                action_type => 'accept',
                result      => 'accepted'
            },
        ]
    },
    result => 'success'
}
```

```
{
    answer => {
        domains => [
            {
                id          => 123456,
                action_type => 'yes',
                result      => 'accepted'
            },
            {
                id          => 123457
                action_type => 'refuse',
                result      => 'refused'
            },
            {
                id           => 123458
                action_type  => 'xxx',
                result       => 'action_type has incorrect format or data',
                error_params => {
                    param => 'action_type'
                },
                error_code  => 'PARAMETER_INCORRECT'
            }
        ]
    },
    result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 8.26. Функция: domain/request\_to\_transfer

##### Доступность:

Партнёры c сервис планом "Партнёр 2" или выше

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

Отправка запроса на перенос желаемых доменных имён, находящихся на обслуживании в REG.RU, на Ваш аккаунт.

##### Доступность:

Партнёры c сервис планом "Партнёр 2" или выше

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| domain\_name | Имя домена, поле не совместимо со спиcком domains. |
| domains | Массив со списком вариантов доменных имён, каждый элемент массива является хешем с ключём dname или domain\_name. Используется только в случае запроса в форматах JSON, XML. |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | Массив со списком хешей, содержащий имена доменов dname и результатом отправки запроса на перенос. Если запрос успешно отправлен, поле result будет иметь значение success, в случае неудачи поле будет содержать текст ошибки. |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/request_to_transfer
input_data={"domains":[{"domain_name":"reg.ru"},{"dname":"test0.ru"},{"dname":"test1.ru"},{"dname":"test2.ru"},{"dname":"test3.ru"}]}
input_format=json
password=test
username=test
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INCORRECT\_DOMAIN\_OWNER\_SERVICE\_PLAN | Domain owner has incorrect service plan or not partner | Владелец домена имеет неподходящий сервис план или не является партнером |
| BELONGS\_TO\_ANOTHER\_REGISTRAR | Domain is belong to another registrar | Домен зарегистрирован у другого регистратора |
| FREE\_DOMAIN | Free domain | Домен не зарегистрирован |
| CHECK\_DOMAIN\_ERROR | Transfer is not possible | Перенос домена невозможен |
| NO\_DOMAINS\_TO\_TRANSFER | Not available to transfer domains | Нет доступных для переноса доменов |
| BAD\_SERVICE\_PLAN | Incorrect partner service plan | Данной услугой разрешено пользоваться только партнерам, имеющим сервис план "Партнер 2" или выше |
| REQUEST\_TO\_TRANSFER\_DISABLED | You are not allowed to use request to transfer | Вам запрещено пользоваться данным сервисом |

Cм. [cтандартные коды ошибок](#common_errors)

## 8.27. Функция: domain/get\_tld\_info

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Запрос на получение информации о зоне

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| tld | Название зоны |

А также [Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| tld | Название зоны |
| email\_verification | Необходима ли верификация почты, 1 -- необходима, 0 -- не требуется |
| idn\_allowed | Флаг разрешения/запрета IDN. 1 -- IDN разрешено, 0 -- IDN запрещено |
| max\_dname\_length | Максимальная длина имени домена |
| min\_dname\_length | Минимальная длина имени домена |
| min\_nss\_count | Минимально разрешенное количество DNS-серверов |
| max\_nss\_count | Максимально разрешенное количество DNS-серверов |
| privacy\_protection\_allowed | Разрешено ли privacy\_protection |
| regperiods | Периоды регистрации |
| reject\_days\_of\_renew\_after\_create | Количество дней с момента регистрации домена в течение которых продление запрещено |
| renew\_grace\_period\_before\_expdate | Количество дней для продления перед expiration\_date |
| renew\_grace\_period\_after\_expdate | Количество дней для продления после expiration\_date |
| renewperiods | Периоды продления |
| restoration\_grace\_period | Количество дней для восстановления после expiration\_date+renew\_grace\_period\_after\_expdate |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/domain/get_tld_info
password=test
tld=ru
username=test
```

##### Примеры успешного ответа:

```
{
   answer => {
      tld_info => {
         email_verification                => '0',
         idn_allowed                       => '0',
         max_dname_length                  => '63',
         max_nss_count                     => '13',
         min_dname_length                  => '2',
         min_nss_count                     => '0',
         privacy_protection_allowed        => '0',
         regperiods                        => '1',
         reject_days_of_renew_after_create => '0',
         renew_grace_period_after_expdate  => '31',
         renew_grace_period_before_expdate => '60',
         renewperiods                      => '1',
         restoration_grace_period          => '0',
         tld                               => 'ru'
      }
   },
   charset => 'utf-8',
   messagestore => null,
   result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| NO\_TLD | tld not given or empty | tld не указано или пустое |
| UNAVAILABLE\_DOMAIN\_ZONE | $tld is unavailable domain zone | $tld не относится к списку поддерживаемых доменных зон |

А также см. [cтандартные коды ошибок](#common_errors)

## 8.28. Функция: domain/send\_email\_verification\_letter

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Отправка писем с запросом верификации почты GTLD доменов и проверка подтверждения email для аккредитованных зон

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| dname | Имя домена |
| check\_only | Флаг запрета отправки письма, только проверка необходимости подтверждения email для зоны |

А также [Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| status | ALREADY\_VERIFIED -- для зоны необходимо подтверждение email, для данного домена email уже подтверждён |
| status | NOT\_VERIFIED -- для зоны необходимо подтверждение email, но для данного домена email ещё не подтверждён |
| status | SENT -- для зоны необходимо подтверждение email, для данного домена email ещё не подтверждён, но письмо уже отправлено |
| status | NOT\_REQUIRED -- для зоны подтверждение email не требуется |

##### Пример запроса:

Запрос информации о необходимости подтверждения email и отправка письма для верификации

```
https://api.reg.ru/api/regru2/domain/send_email_verification_letter
dname=test.com
password=test
username=test
```

Запрос информации о необходимости подтверждения email без отправки письма для верификации

```
https://api.reg.ru/api/regru2/domain/send_email_verification_letter
check_only=1
dname=test.com
password=test
username=test
```

##### Примеры успешного ответа:

```
{
    answer => {
       status => 'ALREADY_VERIFIED'
    },
    charset => 'utf-8',
    messagestore => null,
    result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

См. [cтандартные коды ошибок](#common_errors)

## 8.29. Функция: domain/download\_certificate

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Скачивание сертификата на владение доменом (тип услуги «srv\_voucher»)

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| dname | Имя домена |

А также [Cтандартные параметры идентификации услуги](#common_service_identification_params), [cтандартные параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| status | DOMAIN\_NOT\_FOUND -- домен не найден или не пронадлежит вам |
| status | SERVICE\_NOT\_FOUND -- услуга сертификата для данного домена не найдена |

##### Пример запроса:

Запрос на скачивание сертификата на владение доменом (тип услуги «srv\_voucher»)

```
https://api.reg.ru/api/regru2/domain/download_certificate
dname=test.ru
password=test
username=test
```

##### Примеры успешного ответа:

```
voucher_test.ru.pdf
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

См. [cтандартные коды ошибок](#common_errors)

# 9. Функции для управления DNS-зоной

## 9.1. Функция: zone/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

для тестирования, позволяет проверить доступность управления DNS-зоной доменов; управление DNS-зоной возможно только если домену прописаны DNS сервера REG.RU: ns1.reg.ru и ns2.reg.ru

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов, где для доменов у которых можно управлять зоной поле result будет иметь значение success, иначе — код ошибки указывающий причину |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/nop
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.2. Функция: zone/add\_alias

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

cвязать поддомен с IP-адресом

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, которому назначается IP-адрес. Чтобы назначить IP-адрес самому домену, передайте значение '@', чтобы назначить IP-адрес всем поддоменам, не обозначенным явно в других записях, передайте '\*'. |
| ipaddr | IP-адрес, назначаемый поддомену. |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Доменам test.ru и test.com надо назначить IP-адрес 111.111.111.111

```
https://api.reg.ru/api/regru2/zone/add_alias
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"@","ipaddr":"111.111.111.111","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.3. Функция: zone/add\_aaaa

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

cвязать поддомен с IPv6-адресом

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, которому назначается IPv6-адрес. Чтобы назначить IP-адрес самому домену, передайте значение '@', чтобы назначить IP-адрес всем поддоменам, не обозначенным явно в других записях, передайте '\*'. |
| ipaddr | IPv6-адрес, назначемый поддомену. |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Доменам test.ru и test.com надо назначить IPv6-адрес aa11::a111:11aa:aaa1:aa1a

```
https://api.reg.ru/api/regru2/zone/add_aaaa
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"@","ipaddr":"aa11::a111:11aa:aaa1:aa1a","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.4. Функция: zone/add\_caa

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

указать правила выпуска SSL сертификатов для поддомена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, для которого задаётся правило |
| flags | Флаги правила в виде числа: 0 или 128 |
| tag | Тег для правила. Одно из: isssue, issuewild, iodef |
| value | Значение записи в зависимости от тега: для "issue" и "issuewild" - доменное имя центра сертификации, которому позволен выпуск сертификатов для поддомена; для "iodef" - email или http(s) URL, куда будет направлена информация о запросах на выпуск сертификатов для поддомена, противоречащих правилам описанным в CAA. |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Разрешить выпуск wildcard-сертификатов для домена test.ru центру сертификации ca.example.com

```
https://api.reg.ru/api/regru2/zone/add_caa
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"}],"subdomain":"@","flags":0,"tag":"issuewild","value":"ca.example.com"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| CAA\_INVALID | CAA record is incorrect | Неверное значение CAA записи |
| FLAGS\_INVALID | flags field is incorrect | Неверное значение для поля flags |
| TAG\_INVALID | tag field is incorrect | Неверное значение для поля tag |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.5. Функция: zone/add\_cname

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

cвязать поддомен с адресом другого домена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, которому назначается адрес |
| canonical\_name | Имя домена, которому назначаются синонимы |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Домены 3-го уровня mail.test.ru и mail.test.com должны быть связаны с mx10.test.ru

```
https://api.reg.ru/api/regru2/zone/add_cname
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"mail","canonical_name":"mx10.test.ru","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| CNAME\_INVALID | Invalid CNAME | Неверное имя для CNAME |
| CNAME\_ANDOTHERDATA | For this CNAME have other data already | Для этого CNAME уже есть другие данные |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.6. Функция: zone/add\_https

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

указать правила для соединения с доменом

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, для которого задаётся правило |
| priority | Приоритет. 0 - AliasMode (аналог CNAME), больше нуля - ServiceMode (endpoint и параметры для соединения с ним) |
| target | Целевой домен, с которым необходимо устанавливать соединение |
| value | Пары ключ=значение, разделенные через пробел. Допустимые ключи: alpn, no-default-alpn, ipv4hint, ipv6hint, port, ech, mandatory, keyNNN (NNN - число от 0 до 65534) |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Указать для домена необходимость устанавливать соединение по протоколу HTTP/3 и набор ipv4 адресов, чтобы не делать дополнительных запросов для получения A записей

```
https://api.reg.ru/api/regru2/zone/add_https
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"}],"subdomain":"@","priority":1,"target":".","value":"alpn=h3 ipv4hint=193.12.11.10,193.12.11.11"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| HTTPS\_INVALID | HTTPS record is incorrect | Неверное значение HTTPS записи |
| HTTPS\_TARGET\_INVALID | target field is incorrect | Неверное значение для поля target |
| PRIORITY\_INVALID | Priority not found or have not digital data | Поле priority не определено или содержит нецифровые данные |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.7. Функция: zone/add\_mx

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

указать почтовый сервер в виде доменного имени или IP-адреса, который будет принимать почту для вашего домена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, которому назначается адрес, по умолчанию подразумевается сам домен, т.е. значение @ |
| priority | приоритет почтового сервера, 0 — высший, 10 — минимальный, значение по умолчанию 0 |
| mail\_server | Имя домена или IP-адрес почтового сервера (желательно вводить имя домена, т.к. не все почтовые сервера могут понимать IP-адрес) |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Назначить доменным зонам test.ru и test.com главными почтовые сервера mail.test.ru и mail.test.com

```
https://api.reg.ru/api/regru2/zone/add_mx
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"@","mail_server":"mail","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| MAILHOST\_INVALID | Error in mail\_server IP address or domain name | Некорректно указан IP адрес или имя домена в поле mail\_server |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.8. Функция: zone/add\_ns

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

передать управление поддоменами на другие DNS-сервера

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, который будет управляться другими DNS-серверами |
| dns\_server | Доменное имя DNS-сервера |
| record\_number | Порядковый номер NS-записи, который будет определять относительное расположение NS-записей для поддомена |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Передать управление доменным зонам tt.test.ru и tt.test.com на DNS-сервер ns1.test.ru

```
https://api.reg.ru/api/regru2/zone/add_ns
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"tt","dns_server":"ns1.test.ru","record_number":"10","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| NSADDR\_INVALID | Invalid DNS-server address | Неверный адрес DNS-сервера |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.9. Функция: zone/add\_txt

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавить произвольную текстовую запись (TXT) для поддомена

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | Имя поддомена, для которого добавляется текстовая запись |
| text | текст, допустимо использовать только алфавитноцифровые символы из набора ASCII |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Добавить комментарии для mail.test.ru и mail.test.com

```
https://api.reg.ru/api/regru2/zone/add_txt
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"subdomain":"mail","text":"testmail","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| TEXT\_NOT\_FOUND | Text for record not found | Текст не найден |
| TEXT\_TOOLONG | Text too long | Превышена допустимая длина строки |
| TEXT\_CONTAINS\_INVALID\_CHARACTERS | Text contains invalid characters | Текст содержит недопустимые символы |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.10. Функция: zone/add\_srv

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавить сервисную запись

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| service | сервис, который будет сопоставлен указанному серверу, например для назначения SIP серверу sip.test.ru по upd протоколу надо прописать *sip.*udp |
| priority | приоритет записи |
| weight | нагрузка, которую могут обработать системы, необязательное поле, значение по умолчанию 0 |
| target | сервер обслуживающий службу |
| port | порт обслуживающий службу |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

Направить обслуживание SIP протокола для звонков на xxx@test.ru и xxx@test.com на сервер sip.test.ru по udp протоколу на 5060 порту

```
https://api.reg.ru/api/regru2/zone/add_srv
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"service":"_sip._udp","priority":"0","port":"5060","target":"sip.test.ru","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| SERVICE\_INVALID | Service not found or have incorrect data | Поле service не найдено или содержит некорректные данные |
| PRIORITY\_INVALID | Priority not found or have not digital data | Поле priority не определено или содержит нецифровые данные |
| WEIGHT\_INVALID | Weight have not digital data | Поле weight содержит нецифровые данные |
| PORT\_INVALID | Port not found or have not digital data | Поле port не определено или содержит нецифровые данные |

## 9.11. Функция: zone/get\_resource\_records

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

получение ресурсных записей зоны для каждого домена

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов, где для доменов, у которых можно управлять зоной поле result, будет иметь значение success, иначе — код ошибки, указывающий причину |
| rrs | Ресурсные записи домена |
| subname | поддомен, для которого создана ресурсная запись: \* для всех поддоменов, кроме указанных явно, @ для самого домена, или конкретное имя поддомена |
| rectype | класс, тип записи: A, AAAA, CNAME, MX, NS и другие |
| state | статус записи: A — активна (**устарело**, присутствует только для совместимости) |
| priority | приоритет записи |
| content | содержимое записи: IP адрес для А, IPv6 адрес для АААА, и др |
| soa | Время жизни кеша зоны |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/get_resource_records
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "rrs" : [
               {
                  "content" : "111.222.111.222",
                  "prio" : "0",
                  "rectype" : "A",
                  "state" : "A",
                  "subname" : "www"
               }
            ],
            "service_id" : "12345",
            "soa" : {
               "minimum_ttl" : "12h",
               "ttl" : "1d"
            }
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "rrs" : [
               {
                  "content" : "111.222.111.222",
                  "prio" : "0",
                  "rectype" : "A",
                  "state" : "A",
                  "subname" : "www"
               }
            ],
            "service_id" : "12346",
            "soa" : {
               "minimum_ttl" : "12h",
               "ttl" : "1d"
            }
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INVALID\_ZONE\_RECORD\_TYPE | Zone record type is invalid | Неверный тип записи зоны |

А так-же см. [cтандартные коды ошибок](#common_errors)

## 9.12. Функция: zone/update\_records

##### Доступность:

Партнёры

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавление и/или удаление нескольких ресурсных записей одним запросом, порядок элементов в передаваемом массиве имеет значение, т.к. одни записи могут быть зависимыми от других, в случае возникновения ошибки в одной из записей из action\_list, последующиее игнорируются.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| action\_list | Массив хешей, где каждый хеш содержит параметры для добавления/удаления ресурсной записи. Класс/тип добавляемой записи указывает поле action, допустимые варианты: add\_alias, add\_aaaa, add\_caa, add\_cname, add\_https, add\_mx, add\_ns, add\_txt, add\_srv, add\_spf, remove\_record. Остальные поля хеша зависят от action и соответствуют функциям [add\_alias](#zone_add_alias), [add\_aaaa](#zone_add_aaaa), [remove\_record](#zone_remove_record) и остальным, описанным ниже.  Для примера, приведённого ниже, структура action\_list будет иметь следующий вид:   ``` action_list => [     {         action          => 'add_alias',         subdomain       => 'www',         ipaddr          => '11.22.33.44'     },     {         action          => 'add_cname',         subdomain       => '@',         canonical_name  => 'www.test.ru'     },     {         action          => 'remove_record',         subdomain       => 'mail',         record_type     => 'MX',         priority        => '10',         content         => 'mail-relay.example.com'     } ] ```  Массив action\_list может быть как общим для всего списка доменов (см. 1-й пример запроса), так и особым для каждого домена в списке (см. 2й пример запроса). |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/update_records
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru","action_list":[{"action":"add_alias","subdomain":"www","ipaddr":"11.22.33.44"},{"action":"add_cname","subdomain":"@","canonical_name":"www.test.ru"}]}],"output_content_type":"plain"}
input_format=json
```

##### Примеры ответов:

Успешный ответ на первый запрос:

```
{
    answer => {
        domains => [
            {
                dname       => 'test.ru',
                service_id  => 12345,
                action_list => [
                    {
                        action => 'add_alias',
                        result => 'success'
                    },
                    {
                        action => 'add_cname',
                        result => 'success'
                    }
                ],
                result      => 'success'
            }
        ]
    },
    result => 'success'
}
```

Ответ на второй запрос, содержащий ошибку:

```
{
    answer => {
        domains => [
            {
                dname        => 'test.ru',
                service_id   => 12345,
                error_params => {
                    action => 'add_alias'
                },
                error_code   => 'INVALID_ACTION',
                error_text   => 'Invalid action: add_alias'
            }
        ]
    },
    result => 'success'
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INVALID\_ACTION | Action is invalid or not found: $action | Действие ошибочно или не найдено: $action |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.13. Функция: zone/update\_soa

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

изменить время жизни кеша для зоны

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| ttl | Время жизни кеша для зоны. Либо число в секундах, либо число с суффиксами m для месяцев, w для недель, d для дней, h для часов |
| minimum\_ttl | Время жизни кеша для негативного ответа на запрос в зонe. Формат поля как и в TTL |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/update_soa
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"ttl":"1d","minimum_ttl":"4h","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| SOA\_RECORD\_INVALID | Invalid time for SOA record | Некорректно указано время для SOA-записи |

Cм. [cтандартные коды ошибок](#common_errors)

## 9.14. Функция: zone/tune\_forwarding

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавить ресурсные записи, необходимые для web-форвардинга

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/tune_forwarding
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.15. Функция: zone/clear\_forwarding

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

удалить ресурсные записи, необходимые для web-форвардинга

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/clear_forwarding
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.16. Функция: zone/tune\_parking

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавить ресурсные записи, необходимые для парковки домена

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/tune_parking
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.17. Функция: zone/clear\_parking

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

удалить ресурсные записи, необходимые для парковки домена

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/clear_parking
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 9.18. Функция: zone/remove\_record

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

удалить ресурсную запись

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| subdomain | поддомен для которого будет удаляться запись, обязательное поле |
| record\_type | класс, тип удаляемой записи, обязательное поле |
| priority | приоритет записи, опциональное поле, значение по умолчанию 0. Неприменимо к запросу удаления А-записи (и аналогичных записей) |
| content | содержимое записи, опциональное поле, при его отсутствии помечаются к удалению все записи, попадающие под условие остальных параметров |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/remove_record
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru","dname":"test.com"}],"subdomain":"@","content":"111.111.111.111","record_type":"A","output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

| Код ошибки | error\_text | Описание |
| --- | --- | --- |
| INVALID\_ZONE\_RECORD\_TYPE | Zone record type is invalid | Неверный тип записи зоны |

А так-же см. [cтандартные коды ошибок](#common_errors)

## 9.19. Функция: zone/clear

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

удалить все ресурсные записи

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/zone/clear
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru","dname":"test.com"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "12346"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 10. Функции для управления DNSSEC

## 10.1. Функция: dnssec/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Для тестирования, позволяет проверить доступность управления DNSSEC доменов

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов, где для доменов у которых можно управлять DNSSEC поле result будет иметь значение success, иначе — код ошибки указывающий причину |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/nop
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.net",
            "domain_name" : "test.net",
            "error_params" : null,
            "error_text" : "This domain doesn't support DNSSEC",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.2. Функция: dnssec/get\_status

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Получение статуса DNSSEC домена

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |
| status | статус DNSSEC домена, одно из значений: enabled (включен), disabled (выключен), updating (обновление статуса происходит в данный момент) |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/get_status
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345",
            "status" : "enabled"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "6789",
            "status" : "updating"
         },
         {
            "dname" : "test.net",
            "domain_name" : "test.net",
            "error_params" : null,
            "error_text" : "This domain doesn't support DNSSEC",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.3. Функция: dnssec/enable

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Включение DNSSEC для домена, использующего DNS сервера REG.RU. Проверить завершение выполнения операции можно используя функцию [get\_status](#dnssec_get_status).

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/enable
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "error_params" : {
               "domain_name" : "test.com"
            },
            "error_text" : "DNSSEC updating for domain already in progress",
            "result" : "error"
         },
         {
            "dname" : "test.net",
            "error_params" : {
               "domain_name" : "test.net"
            },
            "error_text" : "This domain not use REG.RU name services",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.4. Функция: dnssec/disable

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Выключение DNSSEC для домена. Проверить завершение выполнения операции можно используя функцию [get\_status](#dnssec_get_status).

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/disable
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "error_params" : {
               "domain_name" : "test.com"
            },
            "error_text" : "DNSSEC updating for domain already in progress",
            "result" : "error"
         },
         {
            "dname" : "test.net",
            "error_params" : {
               "domain_name" : "test.net"
            },
            "error_text" : "This domain is not activated yet",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.5. Функция: dnssec/renew\_ksk

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Обновление KSK ключа для домена, использующего DNS сервера REG.RU. Проверить завершение выполнения операции можно используя функцию [get\_status](#dnssec_get_status).

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/renew_ksk
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "error_params" : {
               "domain_name" : "test.com"
            },
            "error_text" : "DNSSEC updating for domain already in progress",
            "result" : "error"
         },
         {
            "dname" : "test.net",
            "domain_name" : "test.net",
            "error_params" : null,
            "error_text" : "This domain not use REG.RU name services",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.6. Функция: dnssec/renew\_zsk

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Обновление ZSK ключа для домена, использующего DNS сервера REG.RU. Проверить завершение выполнения операции можно используя функцию [get\_status](#dnssec_get_status).

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/renew_zsk
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "error_params" : {
               "domain_name" : "test.com"
            },
            "error_text" : "DNSSEC updating for domain already in progress",
            "result" : "error"
         },
         {
            "dname" : "test.net",
            "domain_name" : "test.net",
            "error_params" : null,
            "error_text" : "This domain not use REG.RU name services",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.7. Функция: dnssec/get\_records

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Получение списка DNSSEC записей домена. Для доменов на DNS серверах REG.RU это DNS записи DS и DNSKEY, а для остальных доменов — записи DNSSEC, в том виде как они хранятся в родительской зоне.

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |
| records | список записей DNSSEC (для доменов на DNS серверах REG.RU записи с полями DS, DNSKEY; для остальных — в свободном формате) |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/get_records
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru"},{"dname":"test.com"},{"dname":"test.net"}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "records" : [
               {
                  "DNSKEY" : "test.ru. IN DNSKEY 256 3 13 CPaFmHh6q/b1zQbXp4w8J3sXUp+YK6BfxzSam/qgps/i5JSiXSQ4kD5v433qdnbnocSyZIDw9EqiAQUbVE/M4g=="
               },
               {
                  "DNSKEY" : "test.ru. IN DNSKEY 257 3 13 X2ehOZBEVxU6baEa58fQx/6Y+gckDeq85XGFW8o6jWFB19wtv6aqdc8ycpIrQaZ4bSLYM7ZyLPJtP6UOkzslDg==",
                  "DS" : "test.ru. IN DS 55528 13 2 cbb775347bdbdff6e1d832595df06add7e610732db131c0f5d3d295e85ae0bef"
               }
            ],
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "records" : [
               {
                  "alg" : "13",
                  "digest" : "17F3E3F4E7FD813432C8E989BD74728BFE9BCA6B518BE2F1155D9EA352067997",
                  "digtype" : "2",
                  "keytag" : "2371"
               }
            ],
            "result" : "success",
            "service_id" : "6789"
         },
         {
            "dname" : "test.net",
            "domain_name" : "test.net",
            "error_params" : null,
            "error_text" : "This domain doesn't support DNSSEC",
            "result" : "error"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 10.8. Функция: dnssec/add\_keys

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

Передача информации о KSK ключах в родительскую зону. Можно использовать только для доменов НЕ использующих DNS сервера REG.RU. Проверить завершение выполнения операции можно используя функции [get\_status](#dnssec_get_status), [get\_records](#dnssec_get_records).

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| records | список DNSKEY или DS записей, в случае отсутствия параметра попытается получить записи с авторитетного DNS сервера |

А также [стандартные параметры идентификации услуги](#common_service_identification_params), [параметры идентификации списка услуг](#common_service_list_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| domains | список доменов c результатами выполнения запроса |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/dnssec/add_keys
input_data={"username":"test","password":"test","domains":[{"dname":"test.ru", "records":["test.ru. 3600 IN DS 2371 13 2 4508a7798c38867c94091bbf91edaf9e6dbf56da0606c748d3d1d1b2382c1602"]},{"dname":"test.com", "records":["test.com. IN DNSKEY 257 3 13 X2ehOZBEVxU6baEa58fQx/6Y+gckDeq85XGFW8o6jWFB19wtv6aqdc8ycpIrQaZ4bSLYM7ZyLPJtP6UOkzslDg=="]}],"output_content_type":"plain"}
input_format=json
```

##### Пример ответа:

```
{
   "answer" : {
      "domains" : [
         {
            "dname" : "test.ru",
            "result" : "success",
            "service_id" : "12345"
         },
         {
            "dname" : "test.com",
            "result" : "success",
            "service_id" : "6789"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 11. Функции для работы с хостингом

## 11.1. Функция: hosting/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

проверка работоспособности API.

##### Пример запроса:

```
https://api.reg.ru/api/regru2/hosting/nop
output_content_type=plain
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 12. Функции для работы с папками

## 12.1. Функция: folder/nop

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

тестовая функция, можно использовать как средство для проверки существования папки

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Идентифицирует папку, с которой будет совершено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

```
https://api.reg.ru/api/regru2/folder/nop
folder_id=123456
folder_name=test_folder_name
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "id" : "-1",
      "name" : "test_folder_name"
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.2. Функция: folder/create

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

создание папки

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name | Задает название новой папки. |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

создание папки с именем test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/create
folder_name=test_folder_name
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.3. Функция: folder/remove

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

удаление папки

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Идентифицирует папку, с которой будет совершено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

удаление из папки с ID folder\_id

```
https://api.reg.ru/api/regru2/folder/remove
folder_id=123456
output_content_type=plain
password=test
username=test
```

удаление из папки с именем folder\_name

```
https://api.reg.ru/api/regru2/folder/remove
folder_id=123456
folder_name=test_folder_name
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.4. Функция: folder/rename

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

переименование папки.

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Идентифицирует папку, с которой будет совершено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| new\_folder\_name | Задает новое имя папки |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

переименование папки с ID folder\_id

```
https://api.reg.ru/api/regru2/folder/rename
folder_id=123456
folder_name=test_folder_name
new_folder_name=new_test_folder_name
output_content_type=plain
password=test
username=test
```

переименование папки с именем folder\_name

```
https://api.reg.ru/api/regru2/folder/rename
folder_name=test_folder_name
new_folder_name=new_test_folder_name
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "folder_content" : [
         {
            "domain_name" : "test1.ru",
            "service_id" : "1000",
            "servtype" : "domain"
         },
         {
            "domain_name" : "test2.ru",
            "service_id" : "1001",
            "servtype" : "domain"
         }
      ]
   },
   "charset" : "utf-8",
   "messagestore" : "null",
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.5. Функция: folder/get\_services

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Нет

##### Назначение:

получить список услуг в папке

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации папок](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

выдать список услуг в папке test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/get_services
folder_name=test_folder_name
output_content_type=plain
password=test
username=test
```

выдать список услуг в папке с ID 12345

```
https://api.reg.ru/api/regru2/folder/get_services
folder_id=12345
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
"answer" => {
"folder_content" => [
    {
	"domain_name" => "test1.ru",
	"service_id" => "1000",
	"servtype" => "domain"
    },
    {
	"domain_name" => "test2.ru",
	"service_id" => "1001",
	"servtype" => "domain"
    }
],
"result" => "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.6. Функция: folder/add\_services

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Да

##### Назначение:

добавление услуг в папку

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Задает название папки, куда будут добавлены услуги.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| services | Задает список услуг, с которыми будет произведено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| return\_folder\_contents | Если значение этого поля уставновлено в "1", то в ответе системы будет присутствовать список услуг в папке, с которой совершено действие. |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

добавление списка услуг (test1.ru, test2.ru) в папку test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/add_services
input_data={"folder_name":"test_folder_name","services":[{"domain_name":"test1.ru"},{"domain_name":"test2.ru"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

добавление списка услуг (с ID=1000, ID=1001) в папку с ID 12345

```
https://api.reg.ru/api/regru2/folder/add_services
input_data={"folder_id":"12345","services":[{"service_id":"1000"},{"service_id":"1001"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

Без переданного параметра return\_folder\_content

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test1.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         },
         {
            "dname" : "test2.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

С параметром return\_folder\_content=1

```
{
   "answer" : {
      "folder_content" : [
         {
            "domain_name" : "test1.ru",
            "service_id" : "1000"
         },
         {
            "domain_name" : "test2.ru",
            "service_id" : "1001"
         }
      ],
      "services" : [
         {
            "dname" : "test01.ru",
            "result" : "success",
            "service_id" : "123456",
            "servtype" : "domain"
         },
         {
            "dname" : "test11.ru",
            "error_code" : "DOMAIN_NOT_FOUND",
            "error_params" : {
               "domain_name" : "test11.ru"
            },
            "result" : "Domain test11.ru not found or not owned by You"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.7. Функция: folder/remove\_services

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Да

##### Назначение:

удаление услуг из папки

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Задает название папки, откуда будут удалены услуги.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| services | Задает список услуг, с которыми будет произведено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| return\_folder\_contents | Если значение этого поля уставновлено в "1", то в ответе системы будет присутствовать список услуг в папке, с которой совершено действие. |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

удаление списка услуг (test1.ru, test2.ru) из папки test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/remove_services
input_data={"folder_name":"test_folder_name","services":[{"domain_name":"test1.ru"},{"domain_name":"test2.ru"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

удаление списка услуг (с ID=1000, ID=1001) из папки с ID 12345

```
https://api.reg.ru/api/regru2/folder/remove_services
input_data={"folder_id":"12345","services":[{"service_id":"1000"},{"service_id":"1001"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

Без переданного параметра return\_folder\_content

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test1.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         },
         {
            "dname" : "test2.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

С параметром return\_folder\_content=1

```
{
   "answer" : {
      "folder_content" : [
         {
            "domain_name" : "test1.ru",
            "service_id" : "1000"
         },
         {
            "domain_name" : "test2.ru",
            "service_id" : "1001"
         }
      ],
      "services" : [
         {
            "dname" : "test01.ru",
            "result" : "success",
            "service_id" : "123456",
            "servtype" : "domain"
         },
         {
            "dname" : "test11.ru",
            "error_code" : "DOMAIN_NOT_FOUND",
            "error_params" : {
               "domain_name" : "test11.ru"
            },
            "result" : "Domain test11.ru not found or not owned by You"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.8. Функция: folder/replace\_services

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Да

##### Назначение:

перезаписывание услуг в папке (в результате данной операции все услуги в указанной папке удаляются, а услуги, указанные в параметре domain\_name или service\_id, добавляются в папку)

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Задает название папки, куда будут перезаписаны услуги.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| services | Задает список услуг, с которыми будет произведено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| return\_folder\_contents | Если значение этого поля уставновлено в "1", то в ответе системы будет присутствовать список услуг в папке, с которой совершено действие. |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

перезапись списка услуг (test1.ru, test2.ru) в папке test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/replace_services
input_data={"folder_name":"test_folder_name","services":[{"domain_name":"test1.ru"},{"domain_name":"test2.ru"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

перезапись списка услуг (с ID=1000, ID=1001) в папке с ID 12345

```
https://api.reg.ru/api/regru2/folder/replace_services
input_data={"folder_id":"12345","services":[{"service_id":"1000"},{"service_id":"1001"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

Без переданного параметра return\_folder\_content

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test1.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         },
         {
            "dname" : "test2.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

С параметром return\_folder\_content=1

```
{
   "answer" : {
      "folder_content" : [
         {
            "domain_name" : "test1.ru",
            "service_id" : "1000"
         },
         {
            "domain_name" : "test2.ru",
            "service_id" : "1001"
         }
      ],
      "services" : [
         {
            "dname" : "test01.ru",
            "result" : "success",
            "service_id" : "123456",
            "servtype" : "domain"
         },
         {
            "dname" : "test11.ru",
            "error_code" : "DOMAIN_NOT_FOUND",
            "error_params" : {
               "domain_name" : "test11.ru"
            },
            "result" : "Domain test11.ru not found or not owned by You"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 12.9. Функция: folder/move\_services

##### Доступность:

Все

##### Поддержка обработки списка услуг:

Да

##### Назначение:

перенос услуг из одной папки в другую

##### [Поля запроса:](#common_input_params)

| Параметр | Описание |
| --- | --- |
| folder\_name или folder\_id | Задает название папки, откуда будут перенесены услуги.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| new\_folder\_name или new\_folder\_id | Задает название папки, куда будут перенесены услуги  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| services | Задает список услуг, с которыми будет произведено действие.  (см. [стандартные параметры идентификации папок](#common_folder_identification_params)) |
| return\_folder\_contents | Если значение этого поля уставновлено в "1", то в ответе системы будет присутствовать список услуг в папке, с которой совершено действие. |

##### [Поля ответа:](#common_response_parameters)

[стандартные ответы системы](#common_response_parameters)

##### Пример запроса:

перенос списка услуг (test1.ru, test2.ru) из папки test\_folder\_name в папку new\_test\_folder\_name

```
https://api.reg.ru/api/regru2/folder/move_services
input_data={"folder_name":"test_folder_name","new_folder_name":"new_test_folder_name","services":[{"domain_name":"test1.ru"},{"domain_name":"test2.ru"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

перенос списка услуг (с ID=1000, ID=1001) из папки c ID 12345 в папку с ID 1234567

```
https://api.reg.ru/api/regru2/folder/move_services
input_data={"folder_id":"12345","new_folder_id":"1234567","services":[{"service_id":"1000"},{"service_id":"1001"}]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

Без переданного параметра return\_folder\_content

```
{
   "answer" : {
      "services" : [
         {
            "dname" : "test1.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         },
         {
            "dname" : "test2.ru",
            "result" : "success",
            "service_id" : "1000",
            "servtype" : "domain"
         }
      ]
   },
   "result" : "success"
}
```

С параметром return\_folder\_content=1

```
{
   "answer" : {
      "folder_content" : [
         {
            "domain_name" : "test1.ru",
            "service_id" : "1000"
         },
         {
            "domain_name" : "test2.ru",
            "service_id" : "1001"
         }
      ],
      "services" : [
         {
            "dname" : "test01.ru",
            "result" : "success",
            "service_id" : "123456",
            "servtype" : "domain"
         },
         {
            "dname" : "test11.ru",
            "error_code" : "DOMAIN_NOT_FOUND",
            "error_params" : {
               "domain_name" : "test11.ru"
            },
            "result" : "Domain test11.ru not found or not owned by You"
         }
      ]
   },
   "result" : "success"
}
```

##### Пример кода:

[Показать пример](#)

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

# 13. Функции для работы с Магазином доменов

## 13.1. Функция: shop/nop

##### Доступность:

Клиенты

##### Поддержка обработки списка услуг:

Да

##### Назначение:

для тестирования, позволяет проверить существования лота по имени домена получить его id

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) и

| Параметр | Описание |
| --- | --- |
| dname | имя домена |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| lot\_id | идентификатор лота |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/nop
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "lot_id" : "123"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 13.2. Функция: shop/delete\_lot

##### Доступность:

Все

##### Назначение:

удаление лота/лотов

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) и

| Параметр | Описание |
| --- | --- |
| dname | список доменов удаляемых лотов |

##### [Поля ответа:](#common_response_parameters)

отсутствуют

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/delete_lot
input_data={"dname":["domain1.ru","domain2.ru","domain3.ru"]}
input_format=json
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 13.3. Функция: shop/get\_info

##### Доступность:

Все

##### Назначение:

получение информации по лоту

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) и

| Параметр | Описание |
| --- | --- |
| dname | имя домена |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| dname | имя домена |
| dname\_puny | имя домена в punycode |
| price\_type | тип цены: "fixed" (фиксированная цена) или "offer" (договорная цена) |
| start\_price | цена |
| site | флаг, что с доменом продается сайт |
| tm | флаг, что с доменом продается торговая марка |
| description | описание |
| keywords | список ключевых слов |
| category\_ids | список идентификаторов категорий |
| lot\_hits | количество просмотров лота |
| bids\_cnt | количество ставок/предложений по лоту |
| first\_create\_domain | дата первой регистрации |
| creation\_date | дата регистрации |
| lot\_date | дата выставления на продажу |
| yandex\_tic | Яндекс тИц |
| google\_pr | Google PageRank |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/get_info
dname=test-shop-api-1.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "category_ids" : [
         "1",
         "2",
         "3"
      ],
      "description" : "desc",
      "dname" : "test-shop-api-1.ru",
      "dname_puny" : "test-shop-api-1.ru",
      "keywords" : [
         "k1",
         "k2",
         "k3"
      ],
      "price_type" : "fixed",
      "site" : "1",
      "start_price" : "200.00",
      "tm" : "1"
   },
   "charset" : "utf-8",
   "messagestore" : "null",
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 13.4. Функция: shop/get\_lot\_list

##### Доступность:

Все

##### Назначение:

получение списка лотов

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) и

| Параметр | Описание |
| --- | --- |
| pg | какую страницу показывать, по умолчанию - 0 |
| itemsonpage | сколько элементов на странице, по умолчанию - 25, допустимые значения: 25, 50, 100, 200, 500 |
| sortcol | поле для сортировки, принимаемые значения: dname, dname\_length, google\_pr, yandex\_tic, is\_online, creation\_date, lot\_date, lot\_hits, start\_price, lot\_mtime, first\_create\_domain, bids\_cnt, keywords |
| sortorder | тип сортировки, принимаемые значения: ASC, DESC, по умолчанию - ASC |
| keywords | ключевое слово |
| show\_my\_lots | флаг "только мои лоты" |
| yandex\_tic\_from | Яндекс тИЦ от |
| yandex\_tic\_to | Яндекс ТИЦ до |
| google\_pr\_from | Google PageRank от |
| google\_pr\_to | Google PageRank до |
| price\_from | цена от |
| price\_to | цена до |
| creation\_date\_from | дата создания домена от |
| creation\_date\_to | дата создания домена до |
| lot\_date\_from | дата создания лота от |
| lot\_date\_to | дата создания лота до |
| lot\_mtime\_from | дата редактирвоания лота от |
| lot\_mtime\_to | дата редактирования лота до |
| dname\_length\_from | длина домена от |
| dname\_length\_to | длина домена до |

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| lots\_cnt | количество лотов |
| lots | список лотов |
| bids\_cnt | количество ставок/предложений по лоту |
| category\_ids | список идентификаторов категорий |
| creation\_date | дата регистрации |
| dname | имя домена |
| dname\_puny | имя домена в punycode |
| first\_create\_domain | дата первой регистрации |
| google\_pr | Google PageRank |
| is\_online | флаг что домен продается online |
| keywords | список ключевых слов |
| lot\_date | дата выставления на продажу |
| lot\_hits | количество просмотров лота |
| lot\_mtime | количество просмотров лота |
| price\_type | тип цены: "fixed" (фиксированная цена) или "offer" (договорная цена) |
| site | флаг, что с доменом продается сайт |
| start\_price | цена |
| tm | флаг, что с доменом продается торговая марка |
| yandex\_tic | Яндекс тИц |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/get_lot_list
itemsonpage=100
output_content_type=plain
password=test
pg=0
show_my_lots=1
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "lots" : [
         {
            "bids_cnt" : "0",
            "category_ids" : "[ 1, 2, 3 ]",
            "creation_date" : "2010-11-12",
            "dname" : "test-shop-api-1.ru",
            "dname_puny" : "test-shop-api-1.ru",
            "first_create_domain" : "2010-11-12",
            "google_pr" : "0",
            "is_online" : "1",
            "keywords" : "test, shop, api",
            "lot_date" : "2011-12-21 17:40:36",
            "lot_hits" : "null",
            "lot_mtime" : "2016-04-08 13:55:02",
            "price_type" : "fixed",
            "site" : "0",
            "start_price" : "3000.00",
            "tm" : "0",
            "yandex_tic" : "0"
         }
      ],
      "lots_cnt" : "1"
   },
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 13.5. Функция: shop/get\_category\_list

##### Доступность:

Все

##### Назначение:

получение списка категорий

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params)

##### [Поля ответа:](#common_response_parameters)

| Поле | Описание |
| --- | --- |
| id | идентификатор категории |
| category\_name | название категории на русском |
| category\_name\_en | название категории на английском |
| subcategories | список подкатегорий, элементы которого содержат: id, category\_name, category\_name\_en |

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/get_category_list
dname=qqq.ru
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "category_list" : [
         {
            "category_name" : "Культура и искусство",
            "category_name_en" : "Culture and Art",
            "subcategories" : [
               {
                  "category_name" : "Видео",
                  "category_name_en" : "Video",
                  "id" : "4"
               },
               {
                  "category_name" : "Изобразительное искусство",
                  "category_name_en" : "Fine art",
                  "id" : "6"
               },
               {
                  "category_name" : "Кино",
                  "category_name_en" : "Cinema",
                  "id" : "10"
               }
            ]
         },
         {
            "category_name" : "Бизнес",
            "category_name_en" : "Business",
            "subcategories" : [
               {
                  "category_name" : "Бухгалтерское дело",
                  "category_name_en" : "Accounting profession",
                  "id" : "32"
               },
               {
                  "category_name" : "Горное дело и бурение",
                  "category_name_en" : "Mining & Engineering",
                  "id" : "34"
               },
               {
                  "category_name" : "Издательство и полиграфия",
                  "category_name_en" : "Printing and publishing services",
                  "id" : "40"
               }
            ]
         },
         {
            "category_name" : "Компьютеры",
            "category_name_en" : "Computers",
            "subcategories" : [
               {
                  "category_name" : "Безопасность, вирусы, антивирусы",
                  "category_name_en" : "Security, viruses, antivirus programs",
                  "id" : "76"
               },
               {
                  "category_name" : "Бытовая автоматизация",
                  "category_name_en" : "Home automation",
                  "id" : "77"
               },
               {
                  "category_name" : "Графика",
                  "category_name_en" : "Graphics",
                  "id" : "79"
               }
            ]
         }
      ]
   },
   "charset" : "utf-8",
   "messagestore" : "null",
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

## 13.6. Функция: shop/get\_suggested\_tags

##### Доступность:

Все

##### Назначение:

получения списка популярных тегов для лота

##### [Поля запроса:](#common_input_params)

[стандартные параметры идентификации услуги](#common_service_identification_params) и

| Параметр | Описание |
| --- | --- |
| limit | количество тегов, необязательный, по умолчанию 10, максимум 50 |

##### [Поля ответа:](#common_response_parameters)

Список тегов

##### Пример запроса:

```
https://api.reg.ru/api/regru2/shop/get_suggested_tags
limit=25
output_content_type=plain
password=test
username=test
```

##### Пример ответа:

```
{
   "answer" : {
      "tags" : [
         "бизнес",
         "интернет",
         "авто",
         "бренд",
         "туризм"
      ]
   },
   "charset" : "utf-8",
   "messagestore" : "null",
   "result" : "success"
}
```

##### Возможные ошибки:

Cм. [cтандартные коды ошибок](#common_errors)

1. [Главная](/)
2. [Партнёрам](/reseller/)
3. Документация на Рег.API 2


© ООО «РЕГ.РУ»
