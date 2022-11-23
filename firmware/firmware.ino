#include <Arduino.h>
#include "M93Cx6.h"

#define PWR_PIN 9
#define CS_PIN 10
#define SK_PIN 7
#define DI_PIN 11
#define DO_PIN 12
#define ORG_PIN 8

M93Cx6 ep = M93Cx6(PWR_PIN, CS_PIN, SK_PIN, DO_PIN, DI_PIN, ORG_PIN, 200);

static uint8_t cfgChip = 66;
static uint16_t cfgSize = 512;
static uint8_t cfgOrg = 8;
static uint16_t cfgDelay = 200;

void setup()
{
    Serial.begin(57600);
    pinMode(LED_BUILTIN, OUTPUT); // LED
    while (!Serial)
    {
        delay(10); // wait for serial port to connect. Needed for native USB
    }
    delay(250);
    Serial.write("\n");
}

void loop()
{
    handleSerial();
}

static uint8_t bufferLength;       // number of characters currently in the buffer
const uint8_t BUFF_SIZE = 16;      // make it big enough to hold your longest command
static char buffer[BUFF_SIZE + 1]; // +1 allows space for the null terminator

void handleSerial()
{
    if (Serial.available())
    {
        char c = Serial.read();
        if ((c == '\r' && bufferLength == 0))
        {
            help();
            return;
        }

        if ((c == '\r') || (c == '\n'))
        {
            // end-of-line received
            if (bufferLength > 0)
            {
                handleCmd(buffer);
            }
            bufferLength = 0;
            return;
        }

        if (c == 127) // handle backspace during command input
        {
            if (bufferLength > 0)
            {
                bufferLength--;
            }
            return;
        }

        if (bufferLength < BUFF_SIZE)
        {
            buffer[bufferLength++] = c;  // append the received character to the array
            buffer[bufferLength] = 0x00; // append the null terminator
        }
        else
        {
            Serial.write('\a');
        }
    }
}

void handleCmd(char *msg)
{
    switch (msg[0])
    {
    case 'h':
        help();
        return;
    case 's':
        parse(msg);
        Serial.write("\n");
        return;
    case 'w':
    case 'r':
    case 'e':
    case 'p':
    case 'x':
        break;
    case '?':
        settings();
        return;
    default:
        Serial.println("invalid command");
        help();
        return;
    }

    if (bufferLength > 8) // parse long command with set options inline
    {
        parse(msg);
    }

    ledOn();
    ep.powerUp();

    switch (msg[0])
    {
    case 'r':
        read();
        break;
    case 'w':
        write();
        break;
    case 'p':
        printBin();
        break;
    case 'e':
        erase();
        break;
    }

    delay(5);
    ep.powerDown();
    ledOff();
}

static char *cmd;

void parse(char *msg)
{
    char *pos;
    cmd = strtok(msg, ",");

    pos = strtok(NULL, ",");
    cfgChip = atoi(pos);

    pos = strtok(NULL, ",");
    cfgSize = atoi(pos);

    pos = strtok(NULL, ",");
    cfgOrg = atoi(pos);

    pos = strtok(NULL, ",");
    cfgDelay = atoi(pos);

    if (cfgSize == 0)
    {
        Serial.println("\ainvalid size");
        return;
    }

    if (!setChip())
    {
        return;
    }

    if (!setOrg())
    {
        return;
    }

    if (!setDelay())
    {
        return;
    }
}

bool setChip()
{
    switch (cfgChip)
    {
    case 46:
        ep.setChip(M93C46);
        break;
    case 56:
        ep.setChip(M93C56);
        break;
    case 66:
        ep.setChip(M93C66);
        break;
    case 76:
        ep.setChip(M93C76);
        break;
    case 86:
        ep.setChip(M93C86);
        break;
    default:
        Serial.println("\ainvalid chip");
        return false;
    }

    return true;
}

bool setOrg()
{
    switch (cfgOrg)
    {
    case 8:
        ep.setOrg(ORG_8);
        break;
    default:
        Serial.println("\ainvalid org");
        return false;
    }

    return true;
}

bool setDelay()
{
    ep.setPinDelay(cfgDelay);
    return true;
}

void help()
{
    Serial.println("--- eep ---");
    Serial.println("s,<chip>,<size>,<org>,<pin_delay> - Set configuration");
    Serial.println("? - Print current configuration");
    Serial.println("r - Read eeprom");
    Serial.println("w - Initiate write mode");
    Serial.println("e - Erase eeprom");
    Serial.println("p - Hex print eeprom content");
    Serial.println("h - This help");
}

void settings()
{
    Serial.println("--- settings ---");
    Serial.print("chip: ");
    Serial.println(cfgChip);
    Serial.print("size: ");
    Serial.println(cfgSize);
    Serial.print("org: ");
    Serial.println(cfgOrg);
    Serial.print("delay: ");
    Serial.println(cfgDelay);
}

void read()
{
    delay(10);
    uint16_t c = 0;
    for (uint16_t i = 0; i < cfgSize; i++)
    {
        Serial.write(ep.read(i));
    }
    Serial.write("\n");
}

static unsigned long lastData;
void write()
{
    lastData = millis();
    ep.writeEnable();
    Serial.write('\f');
    for (uint16_t i = 0; i < cfgSize; i++)
    {
        while (Serial.available() == 0)
        {
            if ((millis() - lastData) > 1000)
            {
                ep.writeDisable();
                Serial.println("\adata read timeout");
                return;
            }
        }
        ep.write(i, Serial.read());
        lastData = millis();
        Serial.print("\f");
    }
    ep.writeDisable();
    Serial.println("\r\n--- write done ---");
}

void erase()
{
    ep.writeEnable();
    ep.eraseAll();
    ep.writeDisable();
    Serial.println("\aeeprom erased");
}

static uint8_t linePos;
void printBin()
{
    linePos = 0;
    char buf[4];
    ledOn();
    Serial.println("--- Hex dump ---");
    uint16_t c = 0;
    for (uint16_t i = 0; i < cfgSize; i++)
    {
        switch (cfgOrg)
        {
        case 8:
            c = ep.read(i);
            sprintf(buf, "%02X ", c);
            break;
        case 16:
            c = ep.read(i);
            sprintf(buf, "%02X%02X ", c >> 8, c & 0xFF);
            break;
        }
        Serial.print(buf);
        linePos++;
        if (linePos == 24)
        {
            Serial.write("\n");
            linePos = 0;
        }
    }
    Serial.write("\n");
    ledOff();
}

void ledOn()
{
    digitalWrite(LED_BUILTIN, HIGH);
}

void ledOff()
{
    digitalWrite(LED_BUILTIN, LOW);
}
