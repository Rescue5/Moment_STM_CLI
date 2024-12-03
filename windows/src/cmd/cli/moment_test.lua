local tag = 'idle'
local lastTele = {}

local function onConnect()
   local sampleRate = intParam('sampleRate', 10)
   sample(sampleRate)
   --chiller(1, 100)
   tare()
end

-- Проверка, находится ли значение в таблице
local function isInTable(value, tbl)
   if type(tbl) ~= "table" then
      error("Second argument must be a table")
   end
   for _, v in ipairs(tbl) do
      if v == value then
         return true
      end
   end
   return false
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
   print('-- bye --')
end

--------------------------------------------------------------------------------

local function test(t)
   local delay    = intParam('delay', 250)
   local pulseMin = intParam('pulseMin', 1000)
   local pulseMax = intParam('pulseMax', 2000)
   local pulseInc = intParam('pulseInc',   25)
   
   -- Список значений газа
   local throttleValues = {1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000}
   local bigDelay = 1500 -- Удержание

   sleep(1000)

   --
   -- фаза тестирования и снятия телеметрии с каждой 100
   --

   tag = 'acceleration'

   for i = pulseMin, pulseMax, pulseInc do
      throttle(i)
      if isInTable(i, throttleValues) then
         sleep(bigDelay) -- Увеличенная задержка для выбранных значений
      else
         sleep(delay)    -- Обычная задержка
      end
   end

   --end

end

--------------------------------------------------------------------------------

return {
   Test		= test,
   OnConnect	= onConnect,
   OnTelemetry	= onTelemetry,
   OnDisconnect	= onDisconnect,
}
