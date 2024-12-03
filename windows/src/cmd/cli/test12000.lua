local tag = 'idle'
local lastTele = {}

local function onConnect()
   local sampleRate = intParam('sampleRate', 10)
   sample(sampleRate)
   chiller(1, 100)
   tare()
end

local function printTelemetry(t)
   local str = string.format(
      '%dµs r/min %d b/pos %d | %.02fA x %.02fV = %.02fW | %.02f°C %.02f°C | %d %d %d | %d %d %d',
      t.Throttle, t.MotorRPM, t.Brake, t.MotorI, t.MotorU, t.MotorP, t.Temp1, t.Temp2, t.Load1, t.Load2, t.Load3, t.GyroX, t.GyroY, t.GyroZ
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
   print('-- bye --')
end

--------------------------------------------------------------------------------

local function test(t)
   local delay    = intParam('delay', 250)
   local pulseMin = intParam('pulseMin', 1000)
   local pulseMax = intParam('pulseMax', 2000)
   local pulseInc = intParam('pulseInc',   50)

   sleep(1000)

   --
   -- фаза разгона двигателя
   --

   tag = 'acceleration'

   for i = pulseMin, pulseMax, pulseInc do
      throttle(i)
      sleep(delay)
   end

   --
   -- фаза торможения двигателя диском
   --

   tag = 'slowdown'

   brake(1, 6000)

   while lastTele.Brake < 6000 do
      if lastTele.MotorRPM < 6000 then
          print('MotorRPM < 6000 limit!')
          brake(1, 0)
          break
      end
      sleep(10)
   end

   tag = 'maxpower'

   sleep(1000)

end

--------------------------------------------------------------------------------

return {
   Test		= test,
   OnConnect	= onConnect,
   OnTelemetry	= onTelemetry,
   OnDisconnect	= onDisconnect,
}
