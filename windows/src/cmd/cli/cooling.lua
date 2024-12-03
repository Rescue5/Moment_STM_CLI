local tag = 'idle'
local lastTele = {}

local function onConnect()
   local sampleRate = intParam('sampleRate', 10)
   sample(sampleRate)
   tare()
end

local function printTelemetry(t)
   local str = string.format(
      'Скорость:%d:Момент:%.02f:Тяга:%d:Об/мин:%d:Ток:%.02f:Напряжение:%.02f:КПД:%.02f',
      t.Throttle, 
      (t.Load2 + t.Load3) / 2, 
      t.Load1, 
      t.MotorRPM, 
      t.MotorI, 
      t.MotorU, 
      t.MotorP
   )
   print(str)
end

local function onTelemetry(t)
   t.Tag = tag
   lastTele = t
   if t.Throttle % 100 == 0 then
      printTelemetry(t)
   end
end

local function onDisconnect()
   throttle(0)
   print('-- Freeze mod disabled --')
end

--------------------------------------------------------------------------------

local function coolingMode()
   local delay = 100 -- Задержка между шагами разгона
   local targetThrottle = 1200 -- Целевое значение газа

   tag = 'cooling'

   -- Включаем вентиляторы на 100%
   chiller(1, 100)

   -- Плавный разгон до 1200
   for i = 1000, targetThrottle, 10 do
      throttle(i)
      sleep(delay)
   end

   -- Удерживаем скорость 1200 бесконечно
   throttle(targetThrottle)
   while true do
      sleep(1000) -- Ждём, пока не прервут выполнение
   end
end

--------------------------------------------------------------------------------

local function test()
   -- Вызов режима охлаждения
   coolingMode()
end

--------------------------------------------------------------------------------

return {
   Test         = test,
   OnConnect    = onConnect,
   OnTelemetry  = onTelemetry,
   OnDisconnect = onDisconnect,
}