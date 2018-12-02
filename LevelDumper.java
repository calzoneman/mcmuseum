import com.mojang.minecraft.level.Level;

import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.ObjectInputStream;

import java.util.zip.GZIPInputStream;
import java.util.zip.GZIPOutputStream;

public class LevelDumper {
    public static void main(String[] args) {
        try (
                FileInputStream inputStream = new FileInputStream(args[0]);
                GZIPInputStream gzipStream = new GZIPInputStream(inputStream)
        ) {
            for (int i = 0; i < 5; i++) gzipStream.read();

            try (ObjectInputStream objectStream = new ObjectInputStream(gzipStream)) {
                Level level = (Level) objectStream.readObject();

                System.out.printf(
                    "size = %dx%dx%d, created=%s, name=%s, creator=%s\n",
                    level.width,
                    level.height,
                    level.depth,
                    new java.util.Date(level.createTime).toString(),
                    level.name,
                    level.creator
                );

                writeLevel(level, args[1]);
            }
        } catch (IOException|ClassNotFoundException exception) {
            throw new RuntimeException(exception);
        }
    }

    private static void writeLevel(Level level, String filename) {
        try (
                FileOutputStream outputStream = new FileOutputStream(filename);
                GZIPOutputStream destStream = new GZIPOutputStream(outputStream)
        ) {
            destStream.write(level.width >> 8);
            destStream.write(level.width & 0xff);
            destStream.write(level.depth >> 8);
            destStream.write(level.depth & 0xff);
            destStream.write(level.height >> 8);
            destStream.write(level.height & 0xff);

            destStream.write(((level.xSpawn << 5) + 16) >> 8);
            destStream.write(((level.xSpawn << 5) + 16) & 0xff);
            destStream.write((level.ySpawn << 5) >> 8);
            destStream.write((level.ySpawn << 5) & 0xff);
            destStream.write(((level.zSpawn << 5) + 16) >> 8);
            destStream.write(((level.zSpawn << 5) + 16) & 0xff);

            destStream.write(level.blocks);
        } catch (IOException exception) {
            throw new RuntimeException(exception);
        }
    }
}
