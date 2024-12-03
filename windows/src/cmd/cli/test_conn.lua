local tag = 'idle'
local lastTele = {}

-- Вызывается при подключении
local function onConnect()
   local sampleRate = intParam('sampleRate', 100000)
   sample(sampleRate)
   tare()
   print('-- Connected successfully --')
end

-- Обработка телеметрии
local function onTelemetry(t)
   t.Tag = tag
   lastTele = t

   -- Отображение успешного подключения через телеметрию
   print(string.format('Соединение установлено'))
end

-- Вызывается при отключении
local function onDisconnect()
   throttle(0)
   print('-- --')
end

-- Пустая функция теста, так как тестирование не требуется
local function test()
   print('-- Test not implemented, only connection verification --')
end

--------------------------------------------------------------------------------

-- Экспортируемые функции
return {
   Test         = test,
   OnConnect    = onConnect,
   OnTelemetry  = onTelemetry,
   OnDisconnect = onDisconnect,
}